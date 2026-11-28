package repos

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"sparkly/analytics/internal/domain"
)

type AnalyzeDeviceRepo struct {
	db driver.Conn
}

func NewAnalyzeDeviceRepo(db driver.Conn) *AnalyzeDeviceRepo {
	return &AnalyzeDeviceRepo{
		db: db,
	}
}

func buildDeviceTimeFilter(deviceID uuid.UUID, filters domain.GetDeviceDetailsFilters) (string, []any) {
	clause := "WHERE device_id = ?"
	args := []any{deviceID}

	if filters.FromTime != nil {
		clause += " AND received_at >= ?"
		args = append(args, filters.FromTime.UTC())
	}
	if filters.ToTime != nil {
		clause += " AND received_at <= ?"
		args = append(args, filters.ToTime.UTC())
	}

	return clause, args
}

type logFrameRow struct {
	PC       uint64  `ch:"pc"`
	Function *string `ch:"function"`
	File     *string `ch:"file"`
	Line     *uint32 `ch:"line"`
}

type logRow struct {
	ParkID     uuid.UUID      `ch:"park_id"`
	DeviceID   uuid.UUID      `ch:"device_id"`
	ReceivedAt time.Time      `ch:"received_at"`
	BootID     sql.NullString `ch:"boot_id"`
	BuildID    sql.NullString `ch:"build_id"`
	Level      sql.NullString `ch:"level"`
	Message    sql.NullString `ch:"message"`
	Frames     []logFrameRow  `ch:"frames"`
}

type logsCountRow struct {
	Info  uint64 `ch:"info"`
	Warn  uint64 `ch:"warn"`
	Error uint64 `ch:"error"`
	Fatal uint64 `ch:"fatal"`
	Trace uint64 `ch:"trace"`
	Debug uint64 `ch:"debug"`
	Total uint64 `ch:"total"`
}

type bootRow struct {
	BootID          sql.NullString `ch:"boot_id"`
	DeviceID        uuid.UUID      `ch:"device_id"`
	ReceivedAt      time.Time      `ch:"received_at"`
	BuildID         sql.NullString `ch:"build_id"`
	Diagnostics     []string       `ch:"diagnostics"`
	ResetReason     string         `ch:"reset_reason"`
	FirmwareVersion string         `ch:"firmware_version"`
	AgentVersion    string         `ch:"agent_version"`
}

type metricRow struct {
	MetricID        uint64      `ch:"metric_id"`
	AvgValue        float64     `ch:"avg_value"`
	MinValue        float64     `ch:"min_value"`
	MaxValue        float64     `ch:"max_value"`
	PointValues     []float64   `ch:"point_values"`
	PointTimestamps []time.Time `ch:"point_timestamps"`
}

func (r *AnalyzeDeviceRepo) logFrameToDomain(f logFrameRow) domain.StackFrame {
	var line uint32
	if f.Line != nil {
		line = *f.Line
	}

	return domain.StackFrame{
		PC:       f.PC,
		Line:     line,
		File:     f.File,
		Function: f.Function,
	}
}

func (r *AnalyzeDeviceRepo) logToDomain(l logRow) (domain.Log, error) {
	var bootID *uuid.UUID
	if l.BootID.Valid {
		inMemBootID, err := uuid.Parse(l.BootID.String)
		if err != nil {
			return domain.Log{}, fmt.Errorf("failed to parse boot_id in log row: %w", err)
		}
		bootID = &inMemBootID
	}

	var buildID *uuid.UUID
	if l.BuildID.Valid {
		inMemBuildID, err := uuid.Parse(l.BuildID.String)
		if err != nil {
			return domain.Log{}, fmt.Errorf("failed to parse build_id in log row: %w", err)
		}
		buildID = &inMemBuildID
	}

	domainFrames := make([]domain.StackFrame, 0, len(l.Frames))
	for _, f := range l.Frames {
		domainFrames = append(domainFrames, r.logFrameToDomain(f))
	}

	return domain.Log{
		ParkID:      l.ParkID,
		DeviceID:    l.DeviceID,
		BootID:      bootID,
		BuildID:     buildID,
		LogLevel:    domain.LogLevelFromString(l.Level.String),
		Message:     l.Message.String,
		Timestamp:   l.ReceivedAt,
		StackFrames: domainFrames,
		Attributes:  make(map[string]string),
	}, nil
}

func (r *AnalyzeDeviceRepo) logsCountToDomain(row logsCountRow) domain.LogsCount {
	return domain.LogsCount{
		Info:  int64(row.Info),
		Warn:  int64(row.Warn),
		Error: int64(row.Error),
		Fatal: int64(row.Fatal),
		Trace: int64(row.Trace),
		Debug: int64(row.Debug),
		Total: int64(row.Total),
	}
}

func (r *AnalyzeDeviceRepo) bootToDomain(b bootRow) (domain.Boot, error) {
	var buildID *uuid.UUID
	if b.BuildID.Valid {
		inMemBuildID, err := uuid.Parse(b.BuildID.String)
		if err != nil {
			return domain.Boot{}, fmt.Errorf("failed to parse build_id in boot row: %w", err)
		}
		buildID = &inMemBuildID
	}

	diagnostics := b.Diagnostics
	if diagnostics == nil {
		diagnostics = make([]string, 0)
	}

	return domain.Boot{
		ID:              b.BootID.String,
		DeviceID:        b.DeviceID,
		ReceivedAt:      b.ReceivedAt,
		BuildID:         buildID,
		Diagnostics:     diagnostics,
		ResetReason:     b.ResetReason,
		FirmwareVersion: b.FirmwareVersion,
		AgentVersion:    b.AgentVersion,
	}, nil
}

func (r *AnalyzeDeviceRepo) metricToDomain(m metricRow) domain.Metric {
	pointsCount := min(len(m.PointValues), len(m.PointTimestamps))
	chartPoints := make([]domain.MetricChartPoint, 0, pointsCount)
	for i := range pointsCount {
		chartPoints = append(chartPoints, domain.MetricChartPoint{
			Value:     m.PointValues[i],
			Timestamp: m.PointTimestamps[i],
		})
	}

	return domain.Metric{
		MetricID:    strconv.FormatUint(m.MetricID, 10),
		AvgValue:    m.AvgValue,
		MinValue:    m.MinValue,
		MaxValue:    m.MaxValue,
		ChartPoints: chartPoints,
	}
}

func (r *AnalyzeDeviceRepo) getDeviceLogs(ctx context.Context, deviceID uuid.UUID, filters domain.GetDeviceDetailsFilters) ([]domain.Log, error) {
	whereClause, args := buildDeviceTimeFilter(deviceID, filters)
	query := fmt.Sprintf(`
		SELECT
			park_id,
			device_id,
			received_at,
			boot_id,
			build_id,
			level,
			message,
			frames
		FROM logs_deduplicated
		%s
		ORDER BY received_at DESC
	`, whereClause)

	var logs []logRow
	if err := r.db.Select(ctx, &logs, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get device logs from db: %w", err)
	}

	domainLogs := make([]domain.Log, 0, len(logs))
	for _, l := range logs {
		domainLog, err := r.logToDomain(l)
		if err != nil {
			return nil, fmt.Errorf("failed to transform log object to domain: %w", err)
		}

		domainLogs = append(domainLogs, domainLog)
	}

	return domainLogs, nil
}

func (r *AnalyzeDeviceRepo) getDeviceLogsCount(ctx context.Context, deviceID uuid.UUID, filters domain.GetDeviceDetailsFilters) (domain.LogsCount, error) {
	whereClause, args := buildDeviceTimeFilter(deviceID, filters)
	query := fmt.Sprintf(`
		SELECT
			countIf(upper(coalesce(level, '')) = 'INFO') AS info,
			countIf(upper(coalesce(level, '')) = 'WARN') AS warn,
			countIf(upper(coalesce(level, '')) = 'ERROR') AS error,
			countIf(upper(coalesce(level, '')) = 'FATAL') AS fatal,
			countIf(upper(coalesce(level, '')) = 'TRACE') AS trace,
			countIf(upper(coalesce(level, '')) = 'DEBUG') AS debug,
			count() AS total
		FROM logs_deduplicated
		%s
	`, whereClause)

	var rows []logsCountRow
	if err := r.db.Select(ctx, &rows, query, args...); err != nil {
		return domain.LogsCount{}, fmt.Errorf("failed to get device logs count from db: %w", err)
	}
	if len(rows) == 0 {
		return domain.LogsCount{}, nil
	}

	return r.logsCountToDomain(rows[0]), nil
}

func (r *AnalyzeDeviceRepo) getDeviceBoots(ctx context.Context, deviceID uuid.UUID, filters domain.GetDeviceDetailsFilters) ([]domain.Boot, error) {
	whereClause, args := buildDeviceTimeFilter(deviceID, filters)
	query := fmt.Sprintf(`
		SELECT
			boot_id,
			device_id,
			received_at,
			build_id,
			diagnostics,
			reset_reason,
			firmware_version,
			agent_version
		FROM boots_deduplicated
		%s
		ORDER BY received_at DESC
	`, whereClause)

	var boots []bootRow
	if err := r.db.Select(ctx, &boots, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get device boots from db: %w", err)
	}

	domainBoots := make([]domain.Boot, 0, len(boots))
	for _, b := range boots {
		domainBoot, err := r.bootToDomain(b)
		if err != nil {
			return nil, fmt.Errorf("failed to transform boot object to domain: %w", err)
		}

		domainBoots = append(domainBoots, domainBoot)
	}

	return domainBoots, nil
}

func (r *AnalyzeDeviceRepo) getDeviceMetrics(ctx context.Context, deviceID uuid.UUID, filters domain.GetDeviceDetailsFilters) ([]domain.Metric, error) {
	whereClause, args := buildDeviceTimeFilter(deviceID, filters)
	query := fmt.Sprintf(`
		SELECT
			metric_id,
			avg(metric_value) AS avg_value,
			min(metric_value) AS min_value,
			max(metric_value) AS max_value,
			groupArray(metric_value) AS point_values,
			groupArray(received_at) AS point_timestamps
		FROM (
			SELECT
				metric_id,
				received_at,
				coalesce(value, toFloat64(counter_delta)) AS metric_value
			FROM metrics_deduplicated
			%s AND (value IS NOT NULL OR counter_delta IS NOT NULL)
			ORDER BY received_at ASC
		)
		GROUP BY metric_id
		ORDER BY metric_id ASC
	`, whereClause)

	var metrics []metricRow
	if err := r.db.Select(ctx, &metrics, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get device metrics from db: %w", err)
	}

	domainMetrics := make([]domain.Metric, 0, len(metrics))
	for _, m := range metrics {
		domainMetrics = append(domainMetrics, r.metricToDomain(m))
	}

	return domainMetrics, nil
}

func (r *AnalyzeDeviceRepo) GetDeviceDetails(ctx context.Context, deviceID uuid.UUID, filters domain.GetDeviceDetailsFilters) (domain.DeviceDetails, error) {
	details := domain.DeviceDetails{
		DeviceID:  deviceID,
		Metrics:   make([]domain.Metric, 0),
		Logs:      make([]domain.Log, 0),
		Boots:     make([]domain.Boot, 0),
		Incidents: make([]domain.Incident, 0),
		Timestamp: time.Now().UTC(),
	}

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		metrics, err := r.getDeviceMetrics(gCtx, deviceID, filters)
		if err != nil {
			return err
		}
		details.Metrics = metrics
		return nil
	})

	g.Go(func() error {
		logs, err := r.getDeviceLogs(gCtx, deviceID, filters)
		if err != nil {
			return err
		}
		details.Logs = logs
		return nil
	})

	g.Go(func() error {
		logsCount, err := r.getDeviceLogsCount(gCtx, deviceID, filters)
		if err != nil {
			return err
		}
		details.LogsCount = logsCount
		return nil
	})

	g.Go(func() error {
		boots, err := r.getDeviceBoots(gCtx, deviceID, filters)
		if err != nil {
			return err
		}
		details.Boots = boots
		return nil
	})

	if err := g.Wait(); err != nil {
		return domain.DeviceDetails{}, fmt.Errorf("failed to get data associated to device from db: %w", err)
	}

	return details, nil
}

type LatestMsgRow struct {
	DeviceID         uuid.UUID `ch:"device_id"`
	LatestReceiveTime time.Time `ch:"latest_receive_time"`
}

func (l LatestMsgRow) ToDomain() domain.DeviceSummary {
	return domain.DeviceSummary{
		DeviceID:  l.DeviceID,
		LastSeenAt: l.LatestReceiveTime,
	}
}

func (r *AnalyzeDeviceRepo) GetDevicesSummary(ctx context.Context, parkID uuid.UUID, filters domain.GetDeviceSummaryFilters) (domain.DevicesSummary, error) {
	whereClause := `park_id = ?`
	args := []any{parkID}

	if filters.GroupIDs != nil && len(*filters.GroupIDs) != 0 {
		whereClause += ` AND hasAny(group_ids, CAST(? AS Array(UUID)))`
		args = append(args, *filters.GroupIDs)
	}

	query := fmt.Sprintf(`
		SELECT
			device_id,
			MAX(received_at) as latest_receive_time
		FROM raw_batches
		WHERE %s
		GROUP BY device_id
	`, whereClause)

	var rows []LatestMsgRow
	if err := r.db.Select(ctx, &rows, query, args...); err != nil {
		return domain.DevicesSummary{}, fmt.Errorf("failed to get latest devices messages from db: %w", err)
	}

	domainDevicesSummary := make([]domain.DeviceSummary, 0, len(rows))
	for _, row := range rows {
		domainDevicesSummary = append(domainDevicesSummary, row.ToDomain())
	}

	return domain.DevicesSummary{
		Devices:   domainDevicesSummary,
		Timestamp: time.Now(),
	}, nil
}
