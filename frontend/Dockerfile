FROM node:18-alpine

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm install

# Copy source code
COPY . .

# Build the application
RUN npm run build

# Install serve to run the application
RUN npm install -g serve

# Expose port
EXPOSE 3000

# Set environment variable
ENV REACT_APP_API_URL=http://backend:8080

# Run the application
CMD ["serve", "-s", "build", "-l", "3000"] 