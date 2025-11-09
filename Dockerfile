# 1. Build Typescript
FROM node:24-slim AS builder

WORKDIR /usr/src/app

COPY package*.json ./

RUN npm ci

COPY . .

RUN npm run build

# 2. Copy built files to new bare image
FROM gcr.io/distroless/nodejs24-debian12 AS production
ENV NODE_ENV=production

WORKDIR /usr/src/app

COPY --from=builder /usr/src/app/dist ./dist
COPY --from=builder /usr/src/app/node_modules ./node_modules

CMD ["dist/app.js"]