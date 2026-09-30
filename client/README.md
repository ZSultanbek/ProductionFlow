# ProductionFlow — client

Flutter app for ProductionFlow. **Android and web only** — `ios/`, `macos/`,
`windows/` and `linux/` are deliberately not in this project, and iOS is out of
scope per CLAUDE.md.

This is the skeleton: two routes, the theme, and fixture data. No features yet.

## Run it

```sh
flutter pub get
flutter run -d chrome          # web
flutter run -d <device-id>     # android, see `flutter devices`
```

Nothing calls the backend yet, so it runs with no Go server and no database.

## Build

```sh
flutter build web --release
flutter build apk --release
```

To point a build at a real API:

```sh
flutter run --dart-define=API_BASE_URL=http://10.0.2.2:8080   # android emulator → host
flutter run --dart-define=API_BASE_URL=http://localhost:8080  # web
```

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

`API_BASE_URL` defaults to empty, which means same-origin. The browser calls
`/api/...` on port 3000 and `nginx.conf` proxies it to the `api` service on the
compose network. So the bundle carries no hostname, the same image runs locally
and on the demo host, and there is no CORS configuration anywhere.

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

The widget tests cover both responsive layouts: the phone layout must show a
`NavigationBar`, the desktop layout a `NavigationRail`, and the landing page must
have exactly one `FilledButton` — if a second primary call to action appears,
that test fails on purpose.

## Layout

```
lib/
  main.dart                    ProviderScope + MaterialApp.router, nothing else
  core/
    theme.dart                 the only place colours, type and shape are defined
    router.dart                go_router: "/" and "/app"
    api_client.dart            dio, configured but not called yet
    mock/
      mock_api.dart            loads the fixtures below
      fixtures/*.json          sample data matching docs/api.md exactly
  features/
    landing/landing_page.dart  the public page
    app_shell/app_shell.dart   the product shell — nav, empty bodies
```

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
