#!/usr/bin/env python3
"""
OrbitStream Satellite Simulator
Generates high-throughput telemetry data and sends to Go ingestion service
"""

import argparse
import asyncio
import logging
import time
from dataclasses import dataclass
from datetime import datetime, timezone

import aiohttp

from config import SimulatorConfig
from generators.telemetry_gen import TelemetryGenerator

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


@dataclass
class Satellite:
    """Represents a single satellite"""
    id: str
    generator: TelemetryGenerator

    async def send_telemetry(self, session: aiohttp.ClientSession,
                            config: SimulatorConfig) -> dict:
        """Send a single telemetry point"""
        telemetry = self.generator.generate_telemetry()

        payload = {
            "satellite_id": self.id,
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "battery_charge_percent": telemetry["battery"],
            "storage_usage_mb": telemetry["storage"],
            "signal_strength_dbm": telemetry["signal"]
        }

        try:
            async with session.post(
                f"{config.api_url}/telemetry",
                json=payload,
                timeout=aiohttp.ClientTimeout(total=5)
            ) as response:
                if response.status == 202:
                    return {"status": "success", "satellite": self.id}
                else:
                    return {"status": "error", "code": response.status}
        except Exception as e:
            return {"status": "error", "message": str(e)}

    async def continuous_send(self, session: aiohttp.ClientSession,
                             config: SimulatorConfig,
                             stats: dict):
        """Continuously send telemetry at target rate"""
        interval = 1.0 / config.points_per_second_per_satellite

        while config.running:
            start_time = time.time()

            result = await self.send_telemetry(session, config)

            # Update statistics
            stats["total_sent"] += 1
            if result["status"] == "success":
                stats["success"] += 1
            else:
                stats["errors"] += 1

            # Maintain exact timing
            elapsed = time.time() - start_time
            sleep_time = max(0, interval - elapsed)
            await asyncio.sleep(sleep_time)


class SatelliteSwarm:
    """Manages multiple satellites sending data concurrently"""

    def __init__(self, config: SimulatorConfig):
        self.config = config
        self.satellites: list[Satellite] = []
        self.stats = {
            "total_sent": 0,
            "success": 0,
            "errors": 0,
            "start_time": time.time()
        }

        # Initialize satellites
        for i in range(config.num_satellites):
            sat_id = f"SAT-{i+1:04d}"
            generator = TelemetryGenerator(
                base_battery=100.0,
                base_storage=0.0,
                base_signal=-50.0,
                anomaly_rate=config.anomaly_rate
            )
            self.satellites.append(Satellite(sat_id, generator))

    async def start(self):
        """Start all satellites"""
        logger.info(f"Starting swarm of {len(self.satellites)} satellites")
        logger.info(f"Target throughput: {self.config.num_satellites * self.config.points_per_second_per_satellite} points/sec")

        # Configure aiohttp connector for high concurrency
        connector = aiohttp.TCPConnector(
            limit=self.config.max_connections,
            limit_per_host=self.config.max_connections_per_host,
            keepalive_timeout=30,
            enable_cleanup_closed=True
        )

        timeout = aiohttp.ClientTimeout(
            total=10,
            connect=5,
            sock_read=5
        )

        async with aiohttp.ClientSession(
            connector=connector,
            timeout=timeout
        ) as session:
            # Create tasks for all satellites
            tasks = [
                sat.continuous_send(
                    session,
                    self.config,
                    self.stats
                )
                for sat in self.satellites
            ]

            # Start statistics reporter
            stats_task = asyncio.create_task(self.report_stats())

            # Run all satellite tasks
            await asyncio.gather(*tasks)

            # Cancel stats task
            stats_task.cancel()

    async def report_stats(self):
        """Periodically report throughput statistics"""
        while self.config.running:
            await asyncio.sleep(5)  # Report every 5 seconds

            elapsed = time.time() - self.stats["start_time"]
            throughput = self.stats["total_sent"] / elapsed
            success_rate = (self.stats["success"] / self.stats["total_sent"] * 100
                          if self.stats["total_sent"] > 0 else 0)

            logger.info(
                f"Throughput: {throughput:.0f} pts/sec | "
                f"Total: {self.stats['total_sent']:,} | "
                f"Success: {success_rate:.1f}% | "
                f"Errors: {self.stats['errors']:,}"
            )


async def main():
    """Main entry point"""
    parser = argparse.ArgumentParser(
        description="OrbitStream Satellite Telemetry Simulator"
    )
    parser.add_argument(
        "--satellites", "-s",
        type=int,
        default=100,
        help="Number of satellites to simulate (default: 100)"
    )
    parser.add_argument(
        "--rate", "-r",
        type=int,
        default=100,
        help="Points per second per satellite (default: 100)"
    )
    parser.add_argument(
        "--api-url",
        type=str,
        default="http://localhost:8080",
        help="Go service API URL (default: http://localhost:8080)"
    )
    parser.add_argument(
        "--duration", "-d",
        type=int,
        default=60,
        help="Duration in seconds (default: 60, 0 = infinite)"
    )
    parser.add_argument(
        "--anomaly-rate",
        type=float,
        default=0.01,
        help="Probability of generating anomalous data (default: 0.01 = 1%%)"
    )

    args = parser.parse_args()

    # Create configuration
    config = SimulatorConfig(
        num_satellites=args.satellites,
        points_per_second_per_satellite=args.rate,
        api_url=args.api_url,
        duration_seconds=args.duration,
        max_connections=1000,
        max_connections_per_host=500,
        anomaly_rate=args.anomaly_rate
    )

    logger.info(f"Configuration: {args.satellites} satellites @ {args.rate} pts/sec each")
    logger.info(f"Total target throughput: {args.satellites * args.rate:,} pts/sec")

    # Create and start swarm
    swarm = SatelliteSwarm(config)

    try:
        if config.duration_seconds > 0:
            # Run for specified duration
            await asyncio.wait_for(swarm.start(), timeout=config.duration_seconds)
        else:
            # Run indefinitely
            await swarm.start()
    except KeyboardInterrupt:
        logger.info("Received interrupt signal, shutting down...")
    except TimeoutError:
        logger.info("Duration elapsed, shutting down...")
    finally:
        config.running = False

        # Final statistics
        elapsed = time.time() - swarm.stats["start_time"]
        throughput = swarm.stats["total_sent"] / elapsed
        success_rate = (swarm.stats["success"] / swarm.stats["total_sent"] * 100
                       if swarm.stats["total_sent"] > 0 else 0)

        logger.info("=" * 60)
        logger.info("FINAL STATISTICS")
        logger.info("=" * 60)
        logger.info(f"Duration: {elapsed:.1f} seconds")
        logger.info(f"Total Points Sent: {swarm.stats['total_sent']:,}")
        logger.info(f"Successful: {swarm.stats['success']:,}")
        logger.info(f"Errors: {swarm.stats['errors']:,}")
        logger.info(f"Average Throughput: {throughput:.0f} pts/sec")
        logger.info(f"Success Rate: {success_rate:.2f}%")
        logger.info("=" * 60)


if __name__ == "__main__":
    asyncio.run(main())
