"""
Shared pytest fixtures for satellite simulator tests
"""
from unittest.mock import Mock, patch

import pytest

from config import SimulatorConfig
from generators.telemetry_gen import TelemetryGenerator
from generators.tle_manager import TLEData
from generators.position_calc import PositionData


# Mock TLE data to prevent network calls during tests
MOCK_TLE_DATA = {
    "ISS": TLEData(
        "ISS",
        "1 25544U 98067A   24010.50000000  .00010000  00000-0  18000-3 0  9998",
        "2 25544  51.6416 250.0000 0005000 200.0000 150.0000 15.50000000420000"
    ),
    "NOAA-18": TLEData(
        "NOAA-18",
        "1 28654U 05018A   24010.50000000  .00000000  00000-0  00000-0 0  9998",
        "2 28654  99.1000 200.0000 0014000 100.0000 260.0000 14.10000000450000"
    ),
}


@pytest.fixture
def mock_tle_manager():
    """Mock TLEManager to prevent network calls during tests"""
    mock = Mock()
    mock.load_tle_data.return_value = MOCK_TLE_DATA
    mock.get_satellite_tle.side_effect = lambda name: MOCK_TLE_DATA.get(name.upper())
    mock.get_available_satellites.return_value = list(MOCK_TLE_DATA.keys())
    mock.get_real_satellite_names.return_value = list(MOCK_TLE_DATA.keys())
    return mock


@pytest.fixture
def mock_position_calculator():
    """Mock PositionCalculator to prevent network calls during tests"""
    mock = Mock()
    mock.get_position.return_value = PositionData(
        latitude=40.7128,
        longitude=-74.0060,
        altitude_km=408.5,
        velocity_kmph=27576.5
    )
    mock.get_velocity.return_value = 27576.5
    mock.is_satellite_visible.return_value = True
    mock.get_available_satellites.return_value = list(MOCK_TLE_DATA.keys())
    mock.get_orbital_period.return_value = 92.68  # ISS orbital period in minutes
    return mock


@pytest.fixture
def mock_tle_and_position(mock_tle_manager, mock_position_calculator):
    """Combined fixture that mocks both TLEManager and PositionCalculator"""
    with patch('generators.tle_manager.TLEManager', return_value=mock_tle_manager), \
         patch('generators.position_calc.PositionCalculator', return_value=mock_position_calculator), \
         patch('satellite_sim.TLEManager', return_value=mock_tle_manager), \
         patch('satellite_sim.PositionCalculator', return_value=mock_position_calculator):
        yield mock_tle_manager, mock_position_calculator


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
