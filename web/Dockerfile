# Build stage
FROM node:26-alpine AS builder

WORKDIR /build

# Install dependencies
COPY package*.json ./
RUN npm ci

# Copy source
COPY . .

# NEXT_PUBLIC_* values are inlined into the client bundle at build time.
ARG NEXT_PUBLIC_API_URL=http://localhost:8080
ARG NEXT_PUBLIC_WS_URL=ws://localhost:8080
ARG NEXT_PUBLIC_TURNSTILE_SITE_KEY=
ARG NEXT_PUBLIC_DEMO_ENABLED=false
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL \
    NEXT_PUBLIC_WS_URL=$NEXT_PUBLIC_WS_URL \
    NEXT_PUBLIC_TURNSTILE_SITE_KEY=$NEXT_PUBLIC_TURNSTILE_SITE_KEY \
    NEXT_PUBLIC_DEMO_ENABLED=$NEXT_PUBLIC_DEMO_ENABLED

# Build Next.js app
RUN npm run build

# Production stage
FROM node:26-alpine

WORKDIR /app

# Install production dependencies only
COPY package*.json ./
RUN npm ci --omit=dev

# Copy built app from builder
COPY --from=builder /build/.next ./.next
COPY --from=builder /build/public ./public

# Environment
ENV NODE_ENV=production

# Run as the unprivileged user shipped with the node image
RUN chown -R node:node /app
USER node

# Port
EXPOSE 3000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD node -e "require('http').get('http://localhost:3000/api/health', (r) => {if (r.statusCode !== 200) throw new Error(r.statusCode)})" || exit 1

# Start
CMD ["npm", "run", "start"]
