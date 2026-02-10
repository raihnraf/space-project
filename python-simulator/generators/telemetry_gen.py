import random
from dataclasses import dataclass

import numpy as np


@dataclass
class TelemetryGenerator:
    """
    Generates realistic satellite telemetry data with:
    - Gradual battery drain
    - Storage accumulation
    - Signal fluctuation
    - Occasional anomalies
    """

    base_battery: float = 100.0  # Starting battery percentage
    base_storage: float = 0.0    # Starting storage in MB
    base_signal: float = -50.0   # Base signal in dBm
    anomaly_rate: float = 0.01   # 1% chance of anomaly

    def __post_init__(self):
        # Initialize random walk parameters
        self.battery = self.base_battery
        self.storage = self.base_storage
        self.signal = self.base_signal

        # Random walk parameters
        self.battery_drain_rate = np.random.normal(0.05, 0.01)  # ~0.05% per reading
        self.storage_growth_rate = np.random.normal(10.0, 2.0)   # ~10 MB per reading
        self.signal_volatility = np.random.normal(0.5, 0.1)      # Signal fluctuation

    def generate_telemetry(self) -> dict[str, float]:
        """Generate a single telemetry point"""

        # Decide if this should be an anomaly
        is_anomaly = random.random() < self.anomaly_rate

        if is_anomaly:
            # Generate anomalous data
            return self._generate_anomaly()
        else:
            # Generate normal data with gradual changes
            return self._generate_normal()

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
