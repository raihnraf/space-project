FROM python:3.11-slim

WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    gcc \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements and install
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy source code
COPY . .

# Make the script executable
RUN chmod +x satellite_sim.py

# Run the simulator
CMD ["python", "satellite_sim.py", "--satellites", "100", "--rate", "100"]
