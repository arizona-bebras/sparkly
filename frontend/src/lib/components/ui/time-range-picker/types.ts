import { CalendarDate, getLocalTimeZone, type DateValue } from '@internationalized/date';

export type PresetKey =
  | '5m'
  | '15m'
  | '30m'
  | '1h'
  | '3h'
  | '6h'
  | '12h'
  | '24h'
  | 'today'
  | 'yesterday'
  | '7d';

export interface TimePreset {
  id: PresetKey;
  label: string;
  getRange: () => { from: Date; to: Date };
}

export interface TimeRange {
  from: Date;
  to: Date;
  preset?: PresetKey;
}

export const PRESETS: TimePreset[] = [
  {
    id: '5m',
    label: 'Последние 5 минут',
    getRange: () => ({
      from: new Date(Date.now() - 5 * 60 * 1000),
      to: new Date(),
    }),
  },
  {
    id: '15m',
    label: 'Последние 15 минут',
    getRange: () => ({
      from: new Date(Date.now() - 15 * 60 * 1000),
      to: new Date(),
    }),
  },
  {
    id: '30m',
    label: 'Последние 30 минут',
    getRange: () => ({
      from: new Date(Date.now() - 30 * 60 * 1000),
      to: new Date(),
    }),
  },
  {
    id: '1h',
    label: 'Последний 1 час',
    getRange: () => ({
      from: new Date(Date.now() - 60 * 60 * 1000),
      to: new Date(),
    }),
  },
  {
    id: '3h',
    label: 'Последние 3 часа',
    getRange: () => ({
      from: new Date(Date.now() - 3 * 3600 * 1000),
      to: new Date(),
    }),
  },
  {
    id: '6h',
    label: 'Последние 6 часов',
    getRange: () => ({
      from: new Date(Date.now() - 6 * 3600 * 1000),
      to: new Date(),
    }),
  },
  {
    id: '12h',
    label: 'Последние 12 часов',
    getRange: () => ({
      from: new Date(Date.now() - 12 * 3600 * 1000),
      to: new Date(),
    }),
  },
  {
    id: '24h',
    label: 'Последние 24 часа',
    getRange: () => ({
      from: new Date(Date.now() - 24 * 3600 * 1000),
      to: new Date(),
    }),
  },
  {
    id: 'today',
    label: 'Сегодня',
    getRange: () => {
      const from = new Date();
      from.setHours(0, 0, 0, 0);
      return {
        from,
        to: new Date(),
      };
    },
  },
  {
    id: 'yesterday',
    label: 'Вчера',
    getRange: () => {
      const from = new Date();
      from.setDate(from.getDate() - 1);
      from.setHours(0, 0, 0, 0);
      const to = new Date(from);
      to.setHours(23, 59, 59, 999);
      return {
        from,
        to,
      };
    },
  },
  {
    id: '7d',
    label: 'Последние 7 дней',
    getRange: () => ({
      from: new Date(Date.now() - 7 * 24 * 3600 * 1000),
      to: new Date(),
    }),
  },
];

export function dateToCalendarDate(date: Date): CalendarDate {
  return new CalendarDate(date.getFullYear(), date.getMonth() + 1, date.getDate());
}

export function calendarDateToDate(cd: DateValue): Date {
  return cd.toDate(getLocalTimeZone());
}

export function formatTimeLeadingZero(num: number): string {
  return num < 10 ? `0${num}` : `${num}`;
}

export function getTimeString(date: Date): string {
  return `${formatTimeLeadingZero(date.getHours())}:${formatTimeLeadingZero(date.getMinutes())}`;
}

export function formatDisplayDates(from: Date, to: Date): string {
  const isSameDay =
    from.getFullYear() === to.getFullYear() &&
    from.getMonth() === to.getMonth() &&
    from.getDate() === to.getDate();

  const isSameMonth = from.getFullYear() === to.getFullYear() && from.getMonth() === to.getMonth();

  const isSameYear = from.getFullYear() === to.getFullYear();

  const monthLong = new Intl.DateTimeFormat('ru-RU', { month: 'long' });
  const monthShort = new Intl.DateTimeFormat('ru-RU', { month: 'short' });
  const fullDate = new Intl.DateTimeFormat('ru-RU', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  });

  if (isSameDay) {
    return `${from.getDate()} ${monthLong.format(from)}`;
  }

  if (isSameMonth) {
    return `${from.getDate()} – ${to.getDate()} ${monthLong.format(from)}`;
  }

  if (isSameYear) {
    return `${from.getDate()} ${monthShort.format(from)} – ${to.getDate()} ${monthShort.format(to)}`;
  }

  return `${fullDate.format(from)} – ${fullDate.format(to)}`;
}

export function formatDisplayTimes(from: Date, to: Date): string {
  const timeFmt = new Intl.DateTimeFormat('ru-RU', {
    hour: '2-digit',
    minute: '2-digit',
  });
  return `${timeFmt.format(from)} ➔ ${timeFmt.format(to)}`;
}

export function formatTimeRange(from: Date, to: Date): string {
  const isSameDay =
    from.getFullYear() === to.getFullYear() &&
    from.getMonth() === to.getMonth() &&
    from.getDate() === to.getDate();

  const isSameYear = from.getFullYear() === to.getFullYear();

  const timeFmt = new Intl.DateTimeFormat('ru-RU', {
    hour: '2-digit',
    minute: '2-digit',
  });

  const dateFmt = new Intl.DateTimeFormat('ru-RU', {
    day: 'numeric',
    month: 'short',
  });

  const fullDateFmt = new Intl.DateTimeFormat('ru-RU', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  });

  if (isSameDay) {
    return `${dateFmt.format(from)} ${timeFmt.format(from)} – ${timeFmt.format(to)}`;
  }

  if (isSameYear) {
    return `${dateFmt.format(from)} ${timeFmt.format(from)} – ${dateFmt.format(to)} ${timeFmt.format(to)}`;
  }

  return `${fullDateFmt.format(from)} ${timeFmt.format(from)} – ${fullDateFmt.format(to)} ${timeFmt.format(to)}`;
}
