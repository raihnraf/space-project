"""
Shared pytest fixtures for satellite simulator tests
"""
import pytest
from generators.telemetry_gen import TelemetryGenerator
from config import SimulatorConfig


@pytest.fixture
def telemetry_generator():
    """Create a TelemetryGenerator with default parameters"""
    return TelemetryGenerator(
        base_battery=100.0,
        base_storage=0.0,
        base_signal=-50.0,
        anomaly_rate=0.01
    )


@pytest.fixture
def telemetry_generator_no_anomalies():
    """Create a TelemetryGenerator with zero anomaly rate"""
    return TelemetryGenerator(
        base_battery=100.0,
        base_storage=0.0,
        base_signal=-50.0,
        anomaly_rate=0.0
    )


@pytest.fixture
def telemetry_generator_high_anomaly_rate():
    """Create a TelemetryGenerator with 100% anomaly rate for testing"""
    return TelemetryGenerator(
        base_battery=100.0,
        base_storage=0.0,
        base_signal=-50.0,
        anomaly_rate=1.0
    )


@pytest.fixture
def simulator_config():
    """Create a SimulatorConfig for testing"""
    return SimulatorConfig(
        num_satellites=5,
        points_per_second_per_satellite=10,
        api_url="http://localhost:8080",
        duration_seconds=1,
        max_connections=100,
        max_connections_per_host=50,
        anomaly_rate=0.01
    )
