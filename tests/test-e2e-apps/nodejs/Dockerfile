# Stage 1: Use an official Node.js image as a base.
FROM docker.io/library/node:lts-trixie-slim@sha256:abe7e5a82b61a8ef67b315b85497c9e65173387ccf2f4a029248184b8e020be5

# Set the working directory inside the container
WORKDIR /usr/src/app

# Copy only package.json first to leverage Docker cache
# We only need package.json since package-lock.json is absent
COPY package.json ./

# Install production dependencies using npm install
# This will generate a package-lock.json within the layer based on package.json
# and install only production dependencies.
RUN npm install --only=production

# Copy the rest of the application source code into the working directory
COPY . .

# Set environment variables
ENV NODE_ENV=production

# Set a default port. This can be overridden at runtime
ENV PORT=3000

# Define the command to run the application
CMD ["node", "app.js"]
