FROM node:20-alpine3.17
RUN npm install express  -g --verbose
COPY index.js .

ENTRYPOINT ["node", "index.js"]