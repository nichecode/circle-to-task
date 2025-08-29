#!/bin/bash

# Build and test the converter
echo "🔨 Building circle-to-task..."
go build -o circle-to-task .

if [ $? -ne 0 ]; then
    echo "❌ Build failed"
    exit 1
fi

echo "✅ Build successful"

# Test with example config
echo "🧪 Testing with example config..."
./circle-to-task -input examples/input-config.yml -output examples/output

if [ $? -ne 0 ]; then
    echo "❌ Test failed"
    exit 1
fi

echo "✅ Test successful"
echo "📁 Check examples/output/ for generated files"
echo ""
echo "🚀 Try it yourself:"
echo "   ./circle-to-task -input <your-config.yml> -output <output-dir>"
