# Build from the repository root to use the pnpm workspace and lockfile.
FROM node:22-bookworm-slim AS build
RUN npm install --global pnpm@11.7.0
WORKDIR /app
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY frontend ./frontend
RUN pnpm --filter sparkly-frontend... install --frozen-lockfile
RUN pnpm --dir frontend build

FROM node:22-bookworm-slim
WORKDIR /app/frontend
ENV NODE_ENV=production HOST=0.0.0.0 PORT=3000
# Keep pnpm's symlink targets; some runtime imports are declared as devDependencies.
COPY --from=build --chown=node:node /app/node_modules /app/node_modules
COPY --from=build --chown=node:node /app/frontend/node_modules ./node_modules
COPY --from=build --chown=node:node /app/frontend/package.json ./package.json
COPY --from=build --chown=node:node /app/frontend/build ./build
USER node
EXPOSE 3000
CMD ["node", "build"]
