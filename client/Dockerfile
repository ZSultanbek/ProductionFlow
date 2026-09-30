# syntax=docker/dockerfile:1.7

# --- Flutter SDK ------------------------------------------------------------
# The SDK is installed from the official repo at an exact tag rather than pulled
# from a prebuilt community image: the published Flutter images lag well behind
# (newest pinned tag available is 3.35.0) and cannot satisfy this project's
# `sdk: ^3.13.2`. Keep FLUTTER_VERSION in step with `flutter --version` locally.
#
# This stage is slow once and then cached forever — it only re-runs when
# FLUTTER_VERSION changes.
FROM debian:bookworm-slim AS flutter

ARG FLUTTER_VERSION=3.47.2
ENV PATH="/flutter/bin:/flutter/bin/cache/dart-sdk/bin:${PATH}"
# Keeps pub and flutter from writing to a home directory that does not exist.
ENV PUB_CACHE=/pub-cache

RUN apt-get update \
 && apt-get install -y --no-install-recommends \
      ca-certificates \
      curl \
      git \
      unzip \
      xz-utils \
 && rm -rf /var/lib/apt/lists/*

# --depth 1 on a tag: the version is read from the tag, and none of the history
# is needed to build.
RUN git clone --depth 1 --branch "${FLUTTER_VERSION}" \
      https://github.com/flutter/flutter.git /flutter \
 && git config --global --add safe.directory /flutter

# Separate layer from the clone, so changing anything below does not re-download
# the SDK. `precache --web` fetches the web engine artifacts and nothing else —
# no Android SDK, no Gradle, no NDK. (precache has no --android flag to disable;
# Android artifacts simply are not fetched unless asked for.)
RUN flutter --disable-analytics \
 && flutter config --no-cli-animations \
 && flutter precache --web \
 && flutter --version

# --- dependencies -----------------------------------------------------------
# pubspec on its own layer, same reasoning as go.mod in the backend image.
FROM flutter AS deps

WORKDIR /app
COPY pubspec.yaml pubspec.lock ./
RUN flutter pub get

# --- build ------------------------------------------------------------------
FROM deps AS build

COPY . .

# Empty by default, which means same-origin: the browser calls /api/... and the
# nginx stage below proxies it to the Go service. No CORS, no baked-in hostname,
# so the same image works locally and on the demo host.
ARG API_BASE_URL=""

RUN flutter build web --release --dart-define=API_BASE_URL="${API_BASE_URL}"

# --- serve ------------------------------------------------------------------
# Everything above is discarded. What ships is nginx plus a few MB of static
# files — none of the ~1 GB of SDK used to produce them.
FROM nginx:alpine

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/build/web /usr/share/nginx/html

EXPOSE 80

# busybox wget is present in the nginx alpine image, so this container can
# health-check itself.
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --spider -q http://localhost/ || exit 1

CMD ["nginx", "-g", "daemon off;"]
