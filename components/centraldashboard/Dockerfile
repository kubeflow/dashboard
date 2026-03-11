# Step 1: Builds and tests
FROM node:16.20.2-bullseye AS build

ARG kubeflowversion
ARG commit

ENV BUILD_VERSION=$kubeflowversion
ENV BUILD_COMMIT=$commit
ENV CHROME_BIN=/usr/bin/chromium
ENV PUPPETEER_SKIP_CHROMIUM_DOWNLOAD=true

COPY . /centraldashboard
WORKDIR /centraldashboard

RUN npm ci \
 && npm run build \
 && npm prune --production


# Step 2: Serve static assets with nginx (smaller, secure)
FROM nginx:1.25-alpine AS serve

# Copy built static assets from build stage
COPY --from=build --chown=nginx:nginx /centraldashboard/public /usr/share/nginx/html
COPY --from=build --chown=nginx:nginx /centraldashboard/app.bundle.js /centraldashboard/app.css /usr/share/nginx/html/
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 8082

