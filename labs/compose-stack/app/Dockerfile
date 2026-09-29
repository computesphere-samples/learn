# A simple, correct Dockerfile for a Node app.
FROM node:22-slim

WORKDIR /app

# Copy only the dependency files first, so this layer is cached until they change.
COPY package.json package-lock.json ./
RUN npm ci --omit=dev

# Then copy the code.
COPY server.js ./

# Don't run as root. The node image ships a "node" user for this.
USER node

ENV PORT=3000
EXPOSE 3000
CMD ["node", "server.js"]
