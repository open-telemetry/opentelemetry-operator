# Stage 1: Use an official Node.js image as a base.
FROM node:lts-bookworm

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
