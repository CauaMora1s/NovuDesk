FROM node:22-alpine

RUN npm install -g pnpm@9

WORKDIR /app

COPY apps/web/package.json ./
RUN pnpm install --no-frozen-lockfile

EXPOSE 5173

CMD ["pnpm", "dev", "--host", "0.0.0.0"]
