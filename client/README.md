# ProductionFlow — client

Flutter app for ProductionFlow. **Android and web only** — `ios/`, `macos/`,
`windows/` and `linux/` are deliberately not in this project, and iOS is out of
scope per CLAUDE.md.

Release 1 is in: register and log in, projects, invite links, shooting days,
the call sheet and the Today screen (US-01 to US-05).

## Run it

Start the API first (`make dev` in `backend/`), then:

```sh
flutter pub get
flutter run -d chrome          # web
flutter run -d <device-id>     # android, see `flutter devices`
```

With nothing else set, the app looks for the API at `http://localhost:8080`, or
at `http://10.0.2.2:8080` on the Android emulator (that is how the emulator
reaches your machine). The API allows any `localhost` origin in dev, so
`flutter run -d chrome` works without CORS setup.

To run with **no server at all**, on fixture data:

```sh
flutter run -d chrome --dart-define=USE_MOCK=true
```

Any email and password log in, as the producer of a sample project. Useful for
working on screens, and as a fallback if the demo server is asleep.

## Build

```sh
flutter build web --release
flutter build apk --release
```

To point a build at a particular API:

```sh
flutter run --dart-define=API_BASE_URL=http://192.168.1.20:8080      # a real phone → your machine's LAN address
flutter build web --release --dart-define=API_BASE_URL=https://productionflow-api.onrender.com
```

A release APK only talks `https`. Plain `http` is allowed in debug builds only
(`android/app/src/debug/AndroidManifest.xml`).

## Docker

The whole project — this client, the API, Postgres and migrations — starts with
one command from the **repo root**:

```sh
docker compose up --build     # then http://localhost:3000
```

Docker is for running the stack, not for developing in. Use `flutter run` above
for the edit loop; rebuilding an image per keystroke is no way to work.

### What the image does

`Dockerfile` has four stages, and only the last one ships:

1. **`flutter`** — installs the SDK from the official repo at an exact tag
   (`FLUTTER_VERSION`, currently `3.47.2`), then `flutter precache --web` only.
   No Android SDK, no Gradle, no NDK. Slow once, then cached until the version
   changes.
2. **`deps`** — `pubspec.yaml` + `pubspec.lock` alone, then `flutter pub get`, so
   editing Dart code does not re-resolve packages.
3. **`build`** — `flutter build web --release`.
4. **nginx:alpine** — serves `build/web`. Everything above is discarded; what
   ships is nginx plus a few MB of static files, not the ~1 GB of SDK that
   produced them.

Keep `FLUTTER_VERSION` in step with `flutter --version` locally. The published
community Flutter images are not used on purpose: the newest pinned tag available
is 3.35.0, which cannot satisfy this project's `sdk: ^3.13.2`.

### Why there is no API hostname in the bundle

The Docker build passes an empty `API_BASE_URL`, which means same-origin. The
browser calls `/api/...` on port 3000 and `nginx.conf` proxies it to the `api`
service on the compose network. So the bundle carries no hostname, the same
image runs anywhere, and inside compose no CORS is involved.

`nginx.conf` also does the two things a Flutter web app needs from a server:

- `try_files $uri $uri/ /index.html` — `go_router` uses real paths (`/app`, not
  `/#/app`), so a refresh on `/app` must still serve the app.
- `index.html`, `flutter_service_worker.js` and `version.json` are sent
  `no-cache` while hashed assets get a 30-day immutable cache. Without that split
  a deploy stays invisible until someone hard-reloads.

## Checks

```sh
flutter analyze   # must be clean before a PR
flutter test
```

Or both at once, with the output saved to `.check.log`:

```sh
tool/check.sh
```

What the tests cover:

- `test/app_test.dart` — the whole R1 flow on screen, against a fake backend
  (`test/support/fake_backend.dart`): log in and register, the redirect to login
  and back, Today in its three states, creating a project, the invite link,
  joining, adding a day, the call sheet, and both responsive layouts.
- `test/api_client_test.dart` — the HTTP client against scripted responses: the
  token header, the error envelope, paging, and logging out on a rejected token.
- `test/models_test.dart` — the fixtures parsed by the real `fromJson`.
- `test/dates_test.dart`, `test/routes_test.dart` — small pure functions.

## Layout

```
lib/
  main.dart                    loads the saved session, then ProviderScope + MaterialApp.router
  core/
    theme.dart                 the only place colours, type and shape are defined
    models.dart                the shapes in docs/api.md, with hand-written fromJson
    backend.dart               the Backend interface every screen talks to + ApiException
    api_client.dart            RealApi: Backend over HTTP (dio)
    mock/
      mock_api.dart            MockBackend: Backend from the fixtures below
      fixtures/*.json          sample data matching docs/api.md exactly
    session.dart               who is logged in, and where that is stored
    providers.dart             the backend, the clock, and the data each screen loads
    routes.dart                every address in the app
    router.dart                go_router, with the login redirects
    dates.dart                 formatting for dates and call times (never converting them)
    widgets.dart               loading / error / empty states and other shared pieces
  features/
    landing/                   the public page
    auth/                      log in, create account
    app_shell/                 bottom bar on a phone, rail on a desktop
    today/                     where do I need to be today
    projects/                  my projects, one project, the invite link
    shooting_day/              add a day, the call sheet
    invite/                    where an invite link lands
```

### Three rules worth knowing

- **Screens never call dio.** They ask `backendProvider` for a `Backend`. That is
  what lets tests and `USE_MOCK` swap the server out.
- **Dates and call times are strings**, exactly as the API sends them
  (`"2026-10-04"`, `"07:30"`). They are formatted in `core/dates.dart` and never
  turned into a `DateTime` — that is how a call time ends up five hours off.
- **Nothing navigates after login.** Signing in changes the session; the router
  notices and redirects. Same for logging out.

Features go in `features/<name>/`, one folder each. Anything shared by two
features moves to `core/`.

## Design rules

These are enforced by `core/theme.dart` being the single source of truth. Read
it before styling anything.

- **One seed colour**, a dark brown (`AppTheme.seed`). Light reads warm
  beige/cream, dark reads espresso. Both palettes are designed; neither is a
  fallback. `themeMode` follows the device.
- **No gradients.** Anywhere. Depth comes from the M3 surface ladder —
  `surface` → `surfaceContainerLow` → `surfaceContainer` → `…High` — plus a
  hairline `outlineVariant` border. Cards are elevation 0.
- **No custom shadows.** If something needs to lift, it moves up the surface
  ladder.
- **One accent**, `ColorScheme.tertiary` (ochre), for the single most important
  thing on a screen. Two uses on one screen means one of them is wrong. On the
  landing page it marks the impact badge — the idea the product is built on.
- **Whitespace is a feature.** Use the `AppTheme.space*` scale, not ad-hoc
  numbers. People read this on set, on a phone, in daylight.
- Type is Fraunces (display) + Inter (body), a wide scale so hierarchy is
  obvious without comparing sizes.
- Layouts above and below `AppTheme.breakpoint` (900px) are written separately,
  not one layout that shrinks.

### A note on fonts

`google_fonts` fetches Fraunces and Inter at runtime the first time they are
used, and falls back to the system font offline. If the demo has to work with no
network, download the `.ttf` files into `assets/fonts/`, declare them in
`pubspec.yaml`, and swap the two `GoogleFonts.*TextTheme()` calls in
`core/theme.dart` for `TextTheme` with `fontFamily`.

### A note on a build warning

`flutter build web` prints:

> Expected to find fonts for (MaterialIcons, packages/cupertino_icons/CupertinoIcons)

`cupertino_icons` was removed from `pubspec.yaml` on purpose — it ships iOS-style
glyphs and iOS is out of scope. The framework still references those font
constants internally, so the warning appears; nothing in this app renders a
Cupertino icon, so nothing is missing at runtime. Ignore it, or re-add the
package if the noise is worse than the dependency.
