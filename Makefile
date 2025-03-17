.PHONY: dev build clean

# Start the Wails development server
dev:
	wails dev

# Build the Wails application
build:
	wails build --clean

# Clean the build artifacts
clean:
	rm -rf build

# Run the application (useful for production testing)
run:
	./build/GoDockElm
