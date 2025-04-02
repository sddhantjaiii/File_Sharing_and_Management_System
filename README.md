# File Sharing & Management System

[![Demo](https://img.shields.io/badge/Watch-Demo-red)](https://youtu.be/ezTB_TqruHA)  

## About the Developer

Hi, I'm **Siddhant Jaiswal**, a B.Tech Computer Science student at VIT Bhopal University, graduating in May 2026. I specialize in backend development, scalable systems, and cloud computing. I've led projects that increased customer retention by 40% and reduced administrative workload by 35%. I'm passionate about building efficient and scalable applications, and this project is a demonstration of my backend development skills.

## Project Overview

A full-stack file sharing and management system built with Go, PostgreSQL, Redis, and React.

## Features

- User authentication with JWT
- File upload and management
- File sharing with public URLs
- File search functionality
- Redis caching for better performance
- Background job for expired file cleanup
- Docker support for easy deployment

## Tech Stack

- **Backend:** Go, Gin, PostgreSQL, Redis
- **Frontend:** React, TailwindCSS
- **Storage:** Local file storage
- **Caching:** Redis
- **Authentication:** JWT
- **Deployment:** Docker & Docker Compose

## Prerequisites

- Docker and Docker Compose
- Go 1.21 or later (for local development)
- Node.js 18 or later (for local development)

## Getting Started

1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd file-sharing-system
   ```

2. Create a `.env` file in the backend directory:
   ```env
   DB_HOST=postgres
   DB_USER=postgres
   DB_PASSWORD=postgres
   DB_NAME=filesharing
   DB_PORT=5432
   REDIS_HOST=redis
   REDIS_PORT=6379
   JWT_SECRET=your-secret-key
   ```

3. Run the application using Docker Compose:
   ```bash
   docker-compose up --build
   ```

4. Access the application:
   - Frontend: http://localhost:3000
   - Backend API: http://localhost:8080

## API Endpoints

### Authentication
- `POST /auth/register` - Register a new user
- `POST /auth/login` - Login and get JWT token

### Files
- `POST /files/upload` - Upload a file
- `GET /files` - List user's files
- `GET /files/search?query=<filename>` - Search files
- `GET /files/share/:file_id` - Get share URL for a file
- `DELETE /files/:file_id` - Delete a file

## Development

### Backend Development
1. Navigate to the backend directory:
   ```bash
   cd backend
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Run the server:
   ```bash
   go run main.go
   ```

### Frontend Development
1. Navigate to the frontend directory:
   ```bash
   cd frontend
   ```

2. Install dependencies:
   ```bash
   npm install
   ```

3. Start the development server:
   ```bash
   npm start
   ```

## Project Structure

```
project-root/
│── backend/
│   ├── main.go
│   ├── routes/
│   │   ├── auth.go
│   │   ├── files.go
│   │   ├── models/
│   │   │   ├── user.go
│   │   │   ├── file.go
│   │   ├── utils/
│   │   │   ├── jwt.go
│   │   │   ├── redis.go
│   ├── uploads/
│── frontend/
│   ├── src/
│   │   ├── components/
│   │   ├── pages/
│   │   ├── App.js
│   │   ├── index.js
│── docker-compose.yml
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

