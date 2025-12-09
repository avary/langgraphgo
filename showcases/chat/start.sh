#!/bin/bash

# LangGraphGo Chat - Quick Start Script

echo "🚀 Starting LangGraphGo Chat Server..."
echo ""

# Check if .env exists
if [ ! -f .env ]; then
    echo "⚠️  .env file not found!"
    echo "Creating from template..."
    cp .env.example .env
    echo ""
    echo "📝 Please edit .env and add your OPENAI_API_KEY"
    echo "Then run this script again."
    exit 1
fi

# Check if OPENAI_API_KEY is set
source .env
if [ -z "$OPENAI_API_KEY" ] || [ "$OPENAI_API_KEY" = "your-api-key-here" ]; then
    echo "❌ OPENAI_API_KEY not configured in .env"
    echo ""
    echo "Please edit .env and add your OpenAI API key:"
    echo "  OPENAI_API_KEY=sk-your-actual-key"
    exit 1
fi

# Create sessions directory if it doesn't exist
mkdir -p sessions

# Build the application
echo "🔨 Building application..."
go build -o chat main.go session.go

if [ $? -ne 0 ]; then
    echo "❌ Build failed"
    exit 1
fi

echo "✅ Build successful"
echo ""

# Get port from .env or use default
PORT=${PORT:-8081}

echo "🌐 Server will start at: http://localhost:$PORT"
echo ""
echo "Press Ctrl+C to stop the server"
echo ""
echo "─────────────────────────────────────────"
echo ""

# Export environment variables from .env and run the server
export $(grep -v '^#' .env | xargs)
./chat
