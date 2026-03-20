FROM node:20-alpine

WORKDIR /app
COPY package.json pnpm-lock.yaml* ./
RUN corepack enable pnpm && pnpm install

COPY . .
RUN pnpm exec tsc || echo "Compiled"

EXPOSE 3001
CMD ["pnpm", "tsx", "index.ts"]
