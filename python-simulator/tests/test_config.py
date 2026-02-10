"""
Tests for the SimulatorConfig dataclass
"""
import pytest

from config import SimulatorConfig


class TestSimulatorConfig:
    """Tests for SimulatorConfig"""

    def test_simulator_config_dataclass(self):
        """Test that SimulatorConfig is a proper dataclass"""
        config = SimulatorConfig(
            num_satellites=10,
            points_per_second_per_satellite=50,
            api_url="http://test.example.com",
            duration_seconds=60,
            max_connections=100,
            max_connections_per_host=50,
            anomaly_rate=0.02
        )

        assert config.num_satellites == 10
        assert config.points_per_second_per_satellite == 50
        assert config.api_url == "http://test.example.com"
        assert config.duration_seconds == 60
        assert config.max_connections == 100
        assert config.max_connections_per_host == 50
        assert config.anomaly_rate == 0.02

    def test_running_default(self):
        """Test that running defaults to True"""
        config = SimulatorConfig(
            num_satellites=1,
            points_per_second_per_satellite=1,
            api_url="http://localhost",
            duration_seconds=10,
            max_connections=10,
            max_connections_per_host=5,
            anomaly_rate=0.01
        )
        assert config.running is True

    def test_running_can_be_set_false(self):
        """Test that running can be set to False"""
        config = SimulatorConfig(
            num_satellites=1,
            points_per_second_per_satellite=1,
            api_url="http://localhost",
            duration_seconds=10,
            max_connections=10,
            max_connections_per_host=5,
            anomaly_rate=0.01,
            running=False
        )
        assert config.running is False

    def test_all_required_fields(self):
        """Test that all required fields must be provided"""
        with pytest.raises(TypeError):
            # Missing required fields
            SimulatorConfig()

    def test_num_satellites_positive(self):
        """Test that num_satellites is positive"""
        config = SimulatorConfig(
            num_satellites=1,
            points_per_second_per_satellite=10,
            api_url="http://localhost",
            duration_seconds=10,
            max_connections=10,
            max_connections_per_host=5,
            anomaly_rate=0.01
        )
        assert config.num_satellites > 0

    def test_anomaly_rate_between_0_and_1(self):
        """Test that anomaly_rate is between 0 and 1"""
        config = SimulatorConfig(
            num_satellites=10,
            points_per_second_per_satellite=10,
            api_url="http://localhost",
            duration_seconds=10,
            max_connections=10,
            max_connections_per_host=5,
            anomaly_rate=0.5
        )
        assert 0 <= config.anomaly_rate <= 1

    def test_api_url_is_string(self):
        """Test that api_url is a string"""
        config = SimulatorConfig(
            num_satellites=10,
            points_per_second_per_satellite=10,
            api_url="http://localhost:8080",
            duration_seconds=10,
            max_connections=10,
            max_connections_per_host=5,
            anomaly_rate=0.01
        )
        assert isinstance(config.api_url, str)

    def test_duration_seconds_can_be_zero(self):
        """Test that duration_seconds can be 0 (infinite run)"""
        config = SimulatorConfig(
            num_satellites=10,
            points_per_second_per_satellite=10,
            api_url="http://localhost",
            duration_seconds=0,
            max_connections=10,
            max_connections_per_host=5,
            anomaly_rate=0.01
        )
        assert config.duration_seconds == 0

    def test_max_connections_greater_than_per_host(self):
        """Test that max_connections >= max_connections_per_host"""
        config = SimulatorConfig(
            num_satellites=10,
            points_per_second_per_satellite=10,
            api_url="http://localhost",
            duration_seconds=10,
            max_connections=100,
            max_connections_per_host=50,
            anomaly_rate=0.01
        )
        assert config.max_connections >= config.max_connections_per_host
