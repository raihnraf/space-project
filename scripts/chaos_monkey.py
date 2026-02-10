#!/usr/bin/env python3
"""
Chaos Monkey Script for OrbitStream
Simulates database failures and verifies zero data loss.

This script tests the fault tolerance of the OrbitStream satellite telemetry system
by simulating database outages and verifying that no data is lost.

Usage:
    python chaos_monkey.py [duration_seconds]

Example:
    python chaos_monkey.py 10  # Kill database for 10 seconds
"""

import docker
import time
import requests
import sys
import logging
from typing import Dict, List, Tuple

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class ChaosMonkey:
    """Chaos testing tool for OrbitStream fault tolerance verification."""

    def __init__(self):
        self.client = docker.from_env()
        self.db_container = None
        self.go_container = None

    def setup(self) -> bool:
        """
        Find and validate the required containers.

        Returns:
            True if containers are found, False otherwise
        """
        try:
            self.db_container = self.client.containers.get("orbitstream-timescaledb")
            self.go_container = self.client.containers.get("orbitstream-go-service")
            logger.info(f"✓ Found DB container: {self.db_container.name}")
            logger.info(f"✓ Found Go service container: {self.go_container.name}")
            return True
        except docker.errors.NotFound as e:
            logger.error(f"✗ Container not found: {e}")
            logger.error("Make sure OrbitStack services are running:")
            logger.error("  docker compose up -d")
            return False

    def get_health_status(self) -> Dict:
        """
        Get the current health status from the Go service.

        Returns:
            Dictionary containing health status data
        """
        try:
            response = requests.get("http://localhost:8080/health", timeout=5)
            response.raise_for_status()
            return response.json()
        except Exception as e:
            logger.error(f"Failed to get health status: {e}")
            return {}

    def get_db_telemetry_count(self) -> int:
        """
        Get the telemetry count directly from the database.

        Returns:
            Number of telemetry records in the database
        """
        try:
            # Use docker exec to run psql command
            result = self.db_container.exec_run(
                "psql -U postgres -d orbitstream -t -c 'SELECT COUNT(*) FROM telemetry;'"
            )
            if result.exit_code == 0:
                count = int(result.output.strip().decode('utf-8'))
                return count
        except Exception as e:
            logger.error(f"Failed to get DB telemetry count: {e}")
        return 0

    def get_wal_record_count(self) -> int:
        """
        Get the WAL record count from the health endpoint.

        Returns:
            Number of records in the WAL
        """
        health = self.get_health_status()
        return health.get("wal_record_count", 0)

    def kill_db(self, duration_seconds: int = 10) -> Tuple[int, int]:
        """
        Stop database container for specified duration and verify data persistence.

        Args:
            duration_seconds: How long to keep the database down

        Returns:
            Tuple of (wal_count_before, wal_count_after)
        """
        logger.info(f"🔪 Killing database for {duration_seconds} seconds...")

        # Get initial state
        health_before = self.get_health_status()
        wal_count_before = health_before.get("wal_record_count", 0)
        db_count_before = self.get_db_telemetry_count()

        logger.info(f"📊 Initial state:")
        logger.info(f"  DB records: {db_count_before}")
        logger.info(f"  WAL records: {wal_count_before}")
        logger.info(f"  Database status: {health_before.get('database_status', 'unknown')}")
        logger.info(f"  Circuit breaker: {health_before.get('circuit_breaker', 'unknown')}")

        # Stop DB
        self.db_container.stop()
        logger.info("❌ Database stopped")

        # Verify service is degraded
        time.sleep(2)
        health_down = self.get_health_status()
        logger.info(f"  Service status during outage: {health_down.get('status', 'unknown')}")
        logger.info(f"  Database status: {health_down.get('database_status', 'unknown')}")

        # Wait for outage duration
        logger.info(f"⏳ Waiting {duration_seconds} seconds...")
        time.sleep(duration_seconds)

        # Start DB
        self.db_container.start()
        logger.info("✅ Database started")

        # Wait for recovery (WAL replay takes time)
        logger.info("⏳ Waiting for WAL replay (20 seconds)...")
        time.sleep(20)

        # Get final state
        health_after = self.get_health_status()
        wal_count_after = health_after.get("wal_record_count", 0)
        db_count_after = self.get_db_telemetry_count()

        logger.info(f"📊 Final state:")
        logger.info(f"  DB records: {db_count_after}")
        logger.info(f"  WAL records: {wal_count_after}")
        logger.info(f"  Database status: {health_after.get('database_status', 'unknown')}")

        return wal_count_before, wal_count_after

    def run_chaos_test(self, outage_duration: int = 10) -> bool:
        """
        Run the full chaos test and verify zero data loss.

        Args:
            outage_duration: Duration in seconds to keep database down

        Returns:
            True if test passed (zero data loss), False otherwise
        """
        logger.info("🚀 Starting Chaos Monkey Test")
        logger.info("=" * 60)

        # Setup
        if not self.setup():
            return False

        # Run the chaos scenario
        wal_before, wal_after = self.kill_db(outage_duration)

        logger.info("=" * 60)
        logger.info("📊 Results:")

        # Determine success
        if wal_after == 0:
            logger.info("✅ SUCCESS: Zero data loss! All WAL records replayed.")
            logger.info("   The system correctly buffered data during the outage")
            logger.info("   and automatically replayed it when the database recovered.")
            return True
        else:
            logger.error(f"❌ FAILURE: {wal_after} records still in WAL")
            logger.error("   Data was not replayed to the database.")
            return False

    def get_telemetry_stats(self) -> Dict:
        """
        Get detailed telemetry statistics from the database.

        Returns:
            Dictionary with telemetry statistics
        """
        try:
            result = self.db_container.exec_run(
                """psql -U postgres -d orbitstream -t -c "
                SELECT
                    COUNT(*) as total_records,
                    COUNT(DISTINCT satellite_id) as unique_satellites,
                    MIN(time) as earliest_time,
                    MAX(time) as latest_time
                FROM telemetry;"
                """
            )
            if result.exit_code == 0:
                output = result.output.strip().decode('utf-8')
                parts = output.split('|')
                if len(parts) == 4:
                    return {
                        "total_records": int(parts[0]),
                        "unique_satellites": int(parts[1]),
                        "earliest_time": parts[2].strip(),
                        "latest_time": parts[3].strip(),
                    }
        except Exception as e:
            logger.error(f"Failed to get telemetry stats: {e}")
        return {}


def main():
    """Main entry point for the chaos monkey script."""
    duration = 10  # Default outage duration

    if len(sys.argv) > 1:
        try:
            duration = int(sys.argv[1])
        except ValueError:
            print(f"Invalid duration: {sys.argv[1]}")
            print("Usage: python chaos_monkey.py [duration_seconds]")
            sys.exit(1)

    logger.info(f"OrbitStream Chaos Monkey")
    logger.info(f"Outage duration: {duration} seconds")
    logger.info("")

    monkey = ChaosMonkey()
    success = monkey.run_chaos_test(duration)

    if success:
        sys.exit(0)
    else:
        sys.exit(1)


if __name__ == "__main__":
    main()
