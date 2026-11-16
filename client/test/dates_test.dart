import 'package:flutter_test/flutter_test.dart';
import 'package:productionflow/core/dates.dart';

void main() {
  test('dateKey writes a date the way the API does', () {
    expect(dateKey(DateTime(2026, 10, 4, 23, 59)), '2026-10-04');
    expect(dateKey(DateTime(2026, 1, 9)), '2026-01-09');
  });

  test('dateKey uses the calendar the DateTime carries, without converting it', () {
    // Late evening in Almaty is still the same local day, whatever UTC says.
    final lateLocal = DateTime(2026, 11, 2, 23, 30);
    expect(dateKey(lateLocal), '2026-11-02');
  });

  test('formatShootDate shows weekday, day and month', () {
    expect(formatShootDate('2026-10-04'), 'Sun 4 Oct');
    expect(formatShootDate('2026-11-02'), 'Mon 2 Nov');
    expect(formatShootDate('2028-02-29'), 'Tue 29 Feb');
  });

  test('formatShootDate returns anything unexpected unchanged', () {
    expect(formatShootDate('soon'), 'soon');
    expect(formatShootDate('2026-13-01'), '2026-13-01');
    expect(formatShootDate(''), '');
  });

  test('callTimeKey zero-pads', () {
    expect(callTimeKey(7, 5), '07:05');
    expect(callTimeKey(0, 0), '00:00');
    expect(callTimeKey(23, 59), '23:59');
  });
}
