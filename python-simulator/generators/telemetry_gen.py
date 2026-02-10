import random
from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Optional

import numpy as np

from .position_calc import PositionCalculator, PositionData


@dataclass
class TelemetryGenerator:
    """
    Generates realistic satellite telemetry data with:
    - Gradual battery drain
    - Storage accumulation
    - Signal fluctuation
    - Occasional anomalies
    - Real-time orbital position (if position_calculator is provided)
    """

    base_battery: float = 100.0  # Starting battery percentage
    base_storage: float = 0.0    # Starting storage in MB
    base_signal: float = -50.0   # Base signal in dBm
    anomaly_rate: float = 0.01   # 1% chance of anomaly
    satellite_name: Optional[str] = None  # Real satellite name for position calculation
    position_calculator: Optional[PositionCalculator] = None  # For orbital position calculation

    def __post_init__(self):
        # Initialize random walk parameters
        self.battery = self.base_battery
        self.storage = self.base_storage
        self.signal = self.base_signal

        # Random walk parameters
        self.battery_drain_rate = np.random.normal(0.05, 0.01)  # ~0.05% per reading
        self.storage_growth_rate = np.random.normal(10.0, 2.0)   # ~10 MB per reading
        self.signal_volatility = np.random.normal(0.5, 0.1)      # Signal fluctuation

        # Track last position for smooth interpolation
        self.last_position: Optional[PositionData] = None

    def generate_telemetry(self) -> dict[str, float]:
        """Generate a single telemetry point with position data"""

        # Decide if this should be an anomaly
        is_anomaly = random.random() < self.anomaly_rate

        # Get base telemetry data
        if is_anomaly:
            telemetry = self._generate_anomaly()
        else:
            telemetry = self._generate_normal()

        # Add position data if calculator is available
        if self.position_calculator and self.satellite_name:
            position = self._get_current_position()
            if position:
                telemetry["latitude"] = round(position.latitude, 6)
                telemetry["longitude"] = round(position.longitude, 6)
                telemetry["altitude_km"] = round(position.altitude_km, 2)
                telemetry["velocity_kmph"] = round(position.velocity_kmph, 2)

        return telemetry

    def _get_current_position(self) -> Optional[PositionData]:
        """Get current satellite position based on orbital mechanics"""
        if not self.position_calculator or not self.satellite_name:
            return None

        try:
            timestamp = datetime.now(timezone.utc)
            position = self.position_calculator.get_position(self.satellite_name, timestamp)

            if position:
                self.last_position = position
                return position

            # Fallback to last known position if calculation fails
            return self.last_position

        except Exception:
            # Return last known position on error
            return self.last_position

    def _generate_normal(self) -> dict[str, float]:
        """Generate normal telemetry with realistic trends"""

        # Battery: Gradual drain with small fluctuations
        self.battery -= self.battery_drain_rate
        self.battery += np.random.normal(0, 0.5)  # Small fluctuation
        self.battery = max(0, min(100, self.battery))  # Clamp to [0, 100]

        # Storage: Gradual accumulation
        self.storage += self.storage_growth_rate
        self.storage += np.random.normal(0, 5)  # Small fluctuation
        self.storage = max(0, self.storage)  # No negative storage

        # Signal: Fluctuate around base
        signal_change = np.random.normal(0, self.signal_volatility)
        self.signal += signal_change
        self.signal = max(-120, min(-30, self.signal))  # Typical range for dBm

        # Simulate occasional data transmission (storage cleanup)
        if self.storage > 90000 and random.random() < 0.1:  # 10% chance when > 90GB
            self.storage -= np.random.uniform(5000, 20000)  # Transmit 5-20 GB

        # Simulate occasional battery charging (when in sunlight)
        if self.battery < 30 and random.random() < 0.05:  # 5% chance when < 30%
            self.battery += np.random.uniform(5, 15)  # Charge 5-15%

        # Re-clamp battery after charging to ensure it stays within bounds
        self.battery = max(0, min(100, self.battery))

        return {
            "battery": round(self.battery, 2),
            "storage": round(self.storage, 2),
            "signal": round(self.signal, 2)
        }

    def _generate_anomaly(self) -> dict[str, float]:
        """Generate anomalous telemetry"""

        anomaly_type = random.choice([
            "battery_critical",
            "storage_full",
            "signal_loss",
            "sudden_discharge"
        ])

        battery, storage, signal = 0, 0, 0

        if anomaly_type == "battery_critical":
            battery = random.uniform(0, 10)  # Critically low
            storage = self.storage + random.uniform(0, 100)
            signal = self.signal + random.uniform(-5, 5)
        elif anomaly_type == "storage_full":
            battery = self.battery + random.uniform(-2, 2)
            storage = random.uniform(95000, 100000)  # Near capacity
            signal = self.signal + random.uniform(-5, 5)
        elif anomaly_type == "signal_loss":
            battery = self.battery + random.uniform(-2, 2)
            storage = self.storage + random.uniform(0, 100)
            signal = random.uniform(-120, -110)  # Very weak signal
        else:  # sudden_discharge
            battery = self.battery - random.uniform(20, 40)  # Sudden drop
            storage = self.storage + random.uniform(0, 100)
            signal = self.signal + random.uniform(-5, 5)

        # Clamp all values to valid ranges
        battery = max(0, min(100, battery))
        storage = max(0, storage)
        signal = max(-120, min(-30, signal))

        return {
            "battery": round(battery, 2),
            "storage": round(storage, 2),
            "signal": round(signal, 2)
        }
