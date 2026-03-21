FROM node:20-alpine

WORKDIR /app

# Install pnpm per user instructions
RUN npm install -g pnpm

COPY package.json pnpm-lock.yaml ./
RUN pnpm install

COPY . .

# Build step if there's a tsconfig (assumes tsc)
RUN pnpm exec tsc

EXPOSE 3001
CMD ["node", "dist/index.js"]
