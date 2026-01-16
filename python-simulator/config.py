from dataclasses import dataclass


@dataclass
class SimulatorConfig:
    """Configuration for the satellite simulator"""

    num_satellites: int
    points_per_second_per_satellite: int
    api_url: str
    duration_seconds: int
    max_connections: int
    max_connections_per_host: int
    anomaly_rate: float

    # Runtime state
    running: bool = True
