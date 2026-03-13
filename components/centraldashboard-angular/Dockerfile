# ---------- Builder stage - Backend ----------
FROM node:16.20.2-bullseye AS backend

WORKDIR /usr/src/app

# Copy package files to the container
COPY ./backend/package*.json ./

# Install dependencies
RUN npm cache clean --force \
 && npm ci

# Copy source code
COPY ./backend/ .

# Build the application
RUN npm run build \
 && npm prune --production

# ---------- Builder stage - Frontend Kubeflow Library ----------
FROM node:16.20.2-bullseye AS frontend-kubeflow-lib

#
# WARNING: the ./kubeflow-common-lib folder is downloaded by the make file.
#          only build this Dockerfile from commands in the Makefile to ensure the library is present.
#

WORKDIR /usr/src/app

# Copy package files to the container
COPY ./kubeflow-common-lib/package*.json ./

# Install dependencies
ENV NG_CLI_ANALYTICS "ci"
RUN npm cache clean --force \
 && npm ci

# Copy source code
COPY ./kubeflow-common-lib/projects ./projects
COPY ./kubeflow-common-lib/angular.json .
COPY ./kubeflow-common-lib/tsconfig.json .

# Build the application
RUN npm run build

# ---------- Builder stage - Frontend ----------
FROM node:16.20.2-bullseye AS frontend

ARG kubeflowversion
ARG commit

ENV BUILD_VERSION=$kubeflowversion
ENV BUILD_COMMIT=$commit

WORKDIR /usr/src/app

# Copy package files to the container
COPY ./frontend/package*.json ./

# Install dependencies
ENV NG_CLI_ANALYTICS "ci"
RUN npm cache clean --force \
 && npm ci

# Copy source code
COPY ./frontend/ .

# Copy the built kubeflow library from the previous stage
COPY --from=frontend-kubeflow-lib /usr/src/app/dist/kubeflow/ ./node_modules/kubeflow/

# Build the application
RUN node scripts/replace-string-in-file.js src/environments/environment.prod.ts 'BUILD_VERSION' $BUILD_VERSION \
 && npm run build -- --output-path=./dist/default --configuration=production

# ---------- Production stage ----------
FROM node:16.20.2-alpine AS serve

USER root

RUN apk add --no-cache tini

USER node:node

ENV NODE_ENV=production

WORKDIR /usr/src/app

# Copy built assets from builder stages
COPY --from=backend --chown=node:node /usr/src/app .
COPY --from=frontend --chown=node:node /usr/src/app/dist/default/ ./dist/public/
COPY --from=frontend --chown=node:node /usr/src/app/dist/default/assets/dashboard_lib.bundle.js ./dist/public/
COPY --from=frontend --chown=node:node /usr/src/app/dist/default/assets/dashboard_lib.bundle.js.map ./dist/public/

EXPOSE 8082

ENTRYPOINT ["/sbin/tini", "--" , "npm", "start"]
