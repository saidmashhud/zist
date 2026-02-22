FROM node:22-alpine AS build
WORKDIR /app

COPY package.json ./
RUN npm install

COPY . .
RUN npm run build && npm prune --omit=dev

# ── Runtime ──────────────────────────────────────────────────────────────────
FROM node:22-alpine
WORKDIR /app

COPY --from=build /app/build ./build
COPY --from=build /app/package.json ./
# adapter-node installs production deps at build time inside /build
# but still needs the runtime node_modules
COPY --from=build /app/node_modules ./node_modules

ENV PORT=3000
ENV HOST=0.0.0.0
# ORIGIN is required for SvelteKit SSR fetches to resolve relative URLs.
# In Docker Compose this is set to http://gateway:8000 so that SSR load
# functions route through the gateway's internal hostname.
ENV ORIGIN=http://gateway:8000

EXPOSE 3000
CMD ["node", "build"]
