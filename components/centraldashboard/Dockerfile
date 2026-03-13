# ---------- Builder stage ----------
FROM node:16.20.2-bullseye AS build

ARG kubeflowversion
ARG commit

ENV BUILD_VERSION=$kubeflowversion
ENV BUILD_COMMIT=$commit
ENV CHROME_BIN=/usr/bin/chromium
ENV PUPPETEER_SKIP_CHROMIUM_DOWNLOAD=true

WORKDIR /usr/src/app

# Copy package files to the container
COPY package*.json ./

# Install dependencies
RUN npm cache clean --force \
 && npm ci

# Copy source code
COPY . .

# Build the application
RUN npm run build \
 && npm prune --production

# ---------- Production stage ----------
FROM node:16.20.2-alpine AS serve

USER root

RUN apk add --no-cache tini

USER node:node

ENV NODE_ENV=production

WORKDIR /usr/src/app

# Copy built assets from builder stage
COPY --from=build --chown=node:node /usr/src/app ./

EXPOSE 8082

ENTRYPOINT ["/sbin/tini", "--" , "npm", "start"]
