import { create } from '@bufbuild/protobuf';
import { timestampFromDate } from '@bufbuild/protobuf/wkt';
import { type Issue, IssueSchema, IssueSeverity } from '$proto';

export function getMockIssues(parkId: string): Issue[] {
  return [
    create(IssueSchema, {
      id: 'iss-01a1a701-d111-7440-9a1b-102030405060',
      parkId,
      title: 'Критический сбой питания',
      description: 'Внезапная перезагрузка контроллера из-за падения входного напряжения ниже 9V',
      severity: IssueSeverity.CRITICAL,
      firstAt: timestampFromDate(new Date(Date.now() - 1000 * 60 * 60 * 36)),
      lastAt: timestampFromDate(new Date(Date.now() - 1000 * 60 * 45)),
    }),
    create(IssueSchema, {
      id: 'iss-01a1a702-e222-7551-ab2c-203040506070',
      parkId,
      title: 'Сбой записи на Flash-память',
      description: 'Ошибка контроллера I/O при сохранении локального буфера телеметрии',
      severity: IssueSeverity.ERROR,
      firstAt: timestampFromDate(new Date(Date.now() - 1000 * 60 * 60 * 18)),
      lastAt: timestampFromDate(new Date(Date.now() - 1000 * 60 * 90)),
    }),
    create(IssueSchema, {
      id: 'iss-01a1a703-f333-7662-bc3d-304050607080',
      parkId,
      title: 'Высокая температура процессора',
      description: 'Превышен порог 85°C в замкнутом корпусе терминала',
      severity: IssueSeverity.WARNING,
      firstAt: timestampFromDate(new Date(Date.now() - 1000 * 60 * 60 * 5)),
      lastAt: timestampFromDate(new Date(Date.now() - 1000 * 60 * 15)),
    }),
    create(IssueSchema, {
      id: 'iss-01a1a704-a444-7773-cd4e-405060708090',
      parkId,
      title: 'Таймаут соединения с брокером MQTT',
      description: 'Связь потеряна более 120 секунд, переход в автономный режим накопления',
      severity: IssueSeverity.WARNING,
      resolvedAt: timestampFromDate(new Date(Date.now() - 1000 * 60 * 30)),
      firstAt: timestampFromDate(new Date(Date.now() - 1000 * 60 * 60 * 8)),
      lastAt: timestampFromDate(new Date(Date.now() - 1000 * 60 * 30)),
    }),
    create(IssueSchema, {
      id: 'iss-01a1a705-b555-7884-de5f-506070809001',
      parkId,
      title: 'Обновление прошивки OTA выполнено',
      description: 'Устройство успешно применило обновление до версии v1.4.2',
      severity: IssueSeverity.INFO,
      resolvedAt: timestampFromDate(new Date(Date.now() - 1000 * 60 * 120)),
      firstAt: timestampFromDate(new Date(Date.now() - 1000 * 60 * 180)),
      lastAt: timestampFromDate(new Date(Date.now() - 1000 * 60 * 120)),
    }),
  ];
}

export function getMockIssue(issueId: string, parkId: string = ''): Issue | undefined {
  const issues = getMockIssues(parkId);
  return issues.find((i) => i.id === issueId) || issues[0];
}
