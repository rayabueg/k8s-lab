# syntax=docker/dockerfile:1

# Build always runs on the host's native platform so native binaries
# (esbuild, etc.) work without QEMU. The output is pure HTML/JS/CSS
# and is copied into the target-platform Nginx image below.
FROM --platform=$BUILDPLATFORM node:22-alpine AS build
WORKDIR /app

COPY package.json tsconfig.json vite.config.ts index.html ./
COPY src ./src

RUN npm install

# Build under a subpath so it can be routed at /vite
ENV VITE_BASE=/vite/
RUN npm run build


FROM nginxinc/nginx-unprivileged:1.27-alpine
COPY --from=build /app/dist/ /usr/share/nginx/html/vite/
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 8080
