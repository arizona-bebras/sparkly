import { createClient, createRouterTransport } from '@connectrpc/connect';
import { createConnectTransport } from '@connectrpc/connect-web';
import { create } from '@bufbuild/protobuf';
import {
  timestampFromDate,
  timestampDate,
  type Timestamp,
  TimestampSchema,
} from '@bufbuild/protobuf/wkt';
import {
  BuildService,
  DeviceSchema,
  DeviceService,
  IssueSchema,
  IssueService,
  ParkService,
  ReleaseSchema,
  ReleaseService,
  UserService,
  type Build,
  type Device,
  type Issue,
  type Release,
  IssueSeverity,
} from './gen/backend/proto/service_pb';
import {
  AnalyticsService,
  AnalyzeDeviceResponseSchema,
  LogSchema,
  LogLevel,
  StackFrameSchema,
  type AnalyzeDeviceRequest,
  type AnalyzeDeviceResponse,
  type Log,
} from './gen/analytics/proto/service_pb';

const transport = createConnectTransport({
  baseUrl: '/',
});

function generateMockLogs(deviceId: string, fromDate: Date, toDate: Date): Log[] {
  const fromTime = fromDate.getTime();
  const toTime = toDate.getTime();
  const duration = Math.max(toTime - fromTime, 60_000);

  const templates: Array<{
    level: LogLevel;
    message: string;
    offsetRatio: number;
    stack?: Array<{ file: string; line: number; pc: bigint }>;
    attributes?: Record<string, string>;
  }> = [
    {
      level: LogLevel.INFO,
      message: 'Boot v0.1.0-alpha',
      offsetRatio: 0.03,
      attributes: { chip: 'ESP32-S3', build: '4a2b9f' },
    },
    {
      level: LogLevel.DEBUG,
      message: 'SPI flash 16MB OK',
      offsetRatio: 0.07,
    },
    {
      level: LogLevel.INFO,
      message: 'DRAM 320KB free',
      offsetRatio: 0.12,
    },
    {
      level: LogLevel.DEBUG,
      message: 'WiFi driver start',
      offsetRatio: 0.18,
    },
    {
      level: LogLevel.INFO,
      message: 'AP scan (4 found)',
      offsetRatio: 0.22,
    },
    {
      level: LogLevel.WARN,
      message: 'Low RSSI (-82dBm)',
      offsetRatio: 0.25,
      attributes: { ssid: 'Office-IoT' },
    },
    {
      level: LogLevel.INFO,
      message: 'WiFi connected: IP 192.168.1.105',
      offsetRatio: 0.27,
    },
    {
      level: LogLevel.DEBUG,
      message: 'DNS resolved mqtt.sparkly.local',
      offsetRatio: 0.31,
    },
    {
      level: LogLevel.INFO,
      message: 'TLS 1.3 handshake OK',
      offsetRatio: 0.34,
    },
    {
      level: LogLevel.INFO,
      message: 'MQTT session connected',
      offsetRatio: 0.37,
    },
    {
      level: LogLevel.DEBUG,
      message: 'Subscribed to dev/telemetry',
      offsetRatio: 0.4,
    },
    {
      level: LogLevel.INFO,
      message: 'Sensors I2C calibration',
      offsetRatio: 0.44,
    },
    {
      level: LogLevel.WARN,
      message: 'I2C NACK on 0x68 (MPU6050)',
      offsetRatio: 0.46,
      attributes: { bus: 'i2c0' },
    },
    {
      level: LogLevel.ERROR,
      message: 'Sensor read timeout',
      offsetRatio: 0.48,
      stack: [{ file: 'drivers/sensors/mpu6050.rs', line: 142, pc: BigInt('0x40081a20') }],
    },
    {
      level: LogLevel.INFO,
      message: 'I2C bus recover attempt 1',
      offsetRatio: 0.49,
    },
    {
      level: LogLevel.INFO,
      message: 'Sensor re-initialized OK',
      offsetRatio: 0.51,
    },
    {
      level: LogLevel.DEBUG,
      message: 'SNTP sync: offset +4ms',
      offsetRatio: 0.55,
    },
    {
      level: LogLevel.INFO,
      message: 'Telemetry batch #1 sent',
      offsetRatio: 0.6,
    },
    {
      level: LogLevel.WARN,
      message: 'RAM heap usage > 85%',
      offsetRatio: 0.65,
    },
    {
      level: LogLevel.WARN,
      message: 'Queue buffer 3 dropped',
      offsetRatio: 0.67,
    },
    {
      level: LogLevel.ERROR,
      message: 'TCP socket write error',
      offsetRatio: 0.71,
      stack: [{ file: 'net/transport/tcp.rs', line: 204, pc: BigInt('0x40091210') }],
    },
    {
      level: LogLevel.INFO,
      message: 'TCP reconnecting to broker',
      offsetRatio: 0.73,
    },
    {
      level: LogLevel.INFO,
      message: 'MQTT reconnected',
      offsetRatio: 0.76,
    },
    {
      level: LogLevel.DEBUG,
      message: 'Heartbeat ping ACK (14ms)',
      offsetRatio: 0.81,
    },
    {
      level: LogLevel.FATAL,
      message: 'Panic: Stack canary corrupt',
      offsetRatio: 0.87,
      stack: [{ file: 'kernel/executor.rs', line: 312, pc: BigInt('0x40080110') }],
    },
    {
      level: LogLevel.ERROR,
      message: 'Crash dump saved to flash',
      offsetRatio: 0.89,
    },
    {
      level: LogLevel.WARN,
      message: 'RTC Watchdog Reset triggered',
      offsetRatio: 0.92,
    },
    {
      level: LogLevel.INFO,
      message: 'System reboot v0.1.0-alpha',
      offsetRatio: 0.95,
    },
    {
      level: LogLevel.INFO,
      message: 'Self-test passed',
      offsetRatio: 0.98,
    },
  ];

  return templates.map((tmpl, idx) => {
    const logTime = new Date(fromTime + duration * tmpl.offsetRatio);
    return create(LogSchema, {
      id: `log-${idx + 1}`,
      deviceId,
      parkId: 'mock-park-id',
      groupIds: ['group-1'],
      buildId: '4a2b9f1e8d7c',
      level: tmpl.level,
      message: tmpl.message,
      timestamp: timestampFromDate(logTime),
      stack: tmpl.stack?.map((s) => create(StackFrameSchema, s)) ?? [],
      attributes: tmpl.attributes ?? {},
    });
  });
}

export interface DeviceBoot {
  id: string;
  timestamp: Timestamp;
  buildId: string;
  resetReason: string;
}

export type AnalyzeDeviceResult = AnalyzeDeviceResponse & {
  boots: DeviceBoot[];
};

function generateMockBoots(fromDate: Date, toDate: Date): DeviceBoot[] {
  const fromTime = fromDate.getTime();
  const duration = Math.max(toDate.getTime() - fromTime, 1000);
  const bootTemplates = [
    { offsetRatio: 0.12, buildId: 'v1.2.0', resetReason: 'power_on' },
    { offsetRatio: 0.47, buildId: 'v1.2.0', resetReason: 'software_reset' },
    { offsetRatio: 0.88, buildId: 'v1.2.1', resetReason: 'watchdog' },
  ];

  return bootTemplates.map((b, idx) => ({
    id: `boot-${idx + 1}`,
    timestamp: timestampFromDate(new Date(fromTime + duration * b.offsetRatio)),
    buildId: b.buildId,
    resetReason: b.resetReason,
  }));
}

const mockAnalyticsTransport = createRouterTransport(({ service }) => {
  service(AnalyticsService, {
    async analyzeDevice(req) {
      const now = new Date();
      const fromDate = req.timerange?.fromTime
        ? timestampDate(req.timerange.fromTime)
        : new Date(now.getTime() - 24 * 60 * 60 * 1000);
      const toDate = req.timerange?.toTime ? timestampDate(req.timerange.toTime) : now;

      const logs = generateMockLogs(req.deviceId, fromDate, toDate);

      return create(AnalyzeDeviceResponseSchema, {
        deviceId: req.deviceId,
        parkId: 'mock-park-id',
        buildId: '4a2b9f1e8d7c',
        metrics: [],
        logs,
        logsCount: {
          info: BigInt(logs.filter((l) => l.level === LogLevel.INFO).length),
          warn: BigInt(logs.filter((l) => l.level === LogLevel.WARN).length),
          error: BigInt(logs.filter((l) => l.level === LogLevel.ERROR).length),
          fatal: BigInt(logs.filter((l) => l.level === LogLevel.FATAL).length),
          trace: BigInt(logs.filter((l) => l.level === LogLevel.TRACE).length),
          debug: BigInt(logs.filter((l) => l.level === LogLevel.DEBUG).length),
          total: BigInt(logs.length),
        },
        timestamp: timestampFromDate(now),
        incidents: [],
      });
    },
  });
});

const analyticsClient = createClient(AnalyticsService, mockAnalyticsTransport);

export const client = {
  user: createClient(UserService, transport),
  park: createClient(ParkService, transport),
  device: createClient(DeviceService, transport),
  issue: createClient(IssueService, transport),
  build: createClient(BuildService, transport),
  release: createClient(ReleaseService, transport),
  analytics: new Proxy(analyticsClient, {
    get(target, prop, receiver) {
      if (prop === 'analyzeDevice') {
        return async (
          req: Parameters<typeof analyticsClient.analyzeDevice>[0],
          options?: Parameters<typeof analyticsClient.analyzeDevice>[1],
        ): Promise<AnalyzeDeviceResult> => {
          const res = await target.analyzeDevice(req, options);
          const now = new Date();
          const fromDate = req?.timerange?.fromTime
            ? timestampDate(create(TimestampSchema, req.timerange.fromTime))
            : new Date(now.getTime() - 24 * 60 * 60 * 1000);
          const toDate = req?.timerange?.toTime
            ? timestampDate(create(TimestampSchema, req.timerange.toTime))
            : now;
          return Object.assign(res, {
            boots: generateMockBoots(fromDate, toDate),
          });
        };
      }
      return Reflect.get(target, prop, receiver);
    },
  }) as typeof analyticsClient & {
    analyzeDevice(
      req: Parameters<typeof analyticsClient.analyzeDevice>[0],
      options?: Parameters<typeof analyticsClient.analyzeDevice>[1],
    ): Promise<AnalyzeDeviceResult>;
  },
};

export type { Build, Device, Issue, Release, AnalyzeDeviceRequest, AnalyzeDeviceResponse, Log };
export { IssueSeverity, DeviceSchema, IssueSchema, ReleaseSchema, AnalyticsService, LogLevel };
