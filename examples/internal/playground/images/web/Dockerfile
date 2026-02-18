FROM node:18-bullseye-slim

WORKDIR /app/

COPY package.json /app
RUN npm install

# Ensure linked resources are available at build time
COPY build.mjs /app
RUN --mount=type=secret,id=SST_RESOURCE_MyBucket,env=SST_RESOURCE_MyBucket \
  node build.mjs

COPY index.mjs /app

ENTRYPOINT ["node", "index.mjs"]