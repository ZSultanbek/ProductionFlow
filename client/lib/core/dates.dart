// Dates and call times arrive from the API as plain strings ("2026-10-04",
// "07:30") and are only ever formatted here, never converted between time
// zones. See the note at the top of core/models.dart.

const List<String> _weekdays = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'];
const List<String> _months = [
  'Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun',
  'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec',
];

String _two(int n) => n.toString().padLeft(2, '0');

/// The calendar date of [moment] as the API writes dates: YYYY-MM-DD.
///
/// Uses the year, month and day exactly as [moment] carries them. Pass a local
/// DateTime (DateTime.now()) to get the date on the phone's own calendar.
String dateKey(DateTime moment) {
  return '${moment.year.toString().padLeft(4, '0')}-${_two(moment.month)}-${_two(moment.day)}';
}

/// "2026-10-04" becomes "Sun 4 Oct". Returns the input unchanged if it is not
/// a date, so a surprise from the server shows up as text instead of a crash.
String formatShootDate(String date) {
  final parts = date.split('-');
  if (parts.length != 3) {
    return date;
  }
  final year = int.tryParse(parts[0]);
  final month = int.tryParse(parts[1]);
  final day = int.tryParse(parts[2]);
  if (year == null || month == null || day == null || month < 1 || month > 12) {
    return date;
  }
  // DateTime.utc is used only to ask "which weekday is this calendar date".
  // Building it from the three numbers involves no time zone at all.
  final weekday = DateTime.utc(year, month, day).weekday;
  return '${_weekdays[weekday - 1]} $day ${_months[month - 1]}';
}

/// Hours and minutes as the API writes call times: "07:30".
String callTimeKey(int hour, int minute) => '${_two(hour)}:${_two(minute)}';
