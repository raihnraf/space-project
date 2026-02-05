"""
Tests for the satellite simulator module
"""
import pytest
import asyncio
from unittest.mock import Mock, AsyncMock, patch
from satellite_sim import Satellite, SatelliteSwarm
from generators.telemetry_gen import TelemetryGenerator
from config import SimulatorConfig


class TestSatellite:
    """Tests for the Satellite class"""

    def test_satellite_initialization(self):
        """Test that a Satellite can be initialized"""
        generator = TelemetryGenerator()
        satellite = Satellite(
            id="SAT-0001",
            generator=generator
        )
        assert satellite.id == "SAT-0001"
        assert satellite.generator == generator

    @pytest.mark.asyncio
    async def test_send_telemetry_creates_correct_payload(self):
        """Test that send_telemetry creates the correct payload structure"""
        generator = TelemetryGenerator(
            base_battery=85.5,
            base_storage=45000.0,
            base_signal=-55.0,
            anomaly_rate=0.0
        )
        satellite = Satellite(id="SAT-0001", generator=generator)

        # Create a mock response that works as an async context manager
        mock_response = Mock()
        mock_response.status = 202
        mock_response.json = AsyncMock(return_value={"status": "accepted"})

        # Create an async context manager mock
        class AsyncContextManagerMock:
            async def __aenter__(self):
                return mock_response
            async def __aexit__(self, *args):
                pass

        # Mock the session.post to return the context manager
        mock_session = Mock()
        mock_session.post = Mock(return_value=AsyncContextManagerMock())

        config = SimulatorConfig(
            num_satellites=1,
            points_per_second_per_satellite=1,
            api_url="http://localhost:8080",
            duration_seconds=1,
            max_connections=10,
            max_connections_per_host=5,
            anomaly_rate=0.01
        )
        result = await satellite.send_telemetry(mock_session, config)

        assert result["status"] == "success"
        assert result["satellite"] == "SAT-0001"

    def test_satellite_id_format(self):
        """Test that satellite IDs follow the expected format"""
        generator = TelemetryGenerator()
        satellite = Satellite(id="SAT-1234", generator=generator)
        assert satellite.id.startswith("SAT-")
        assert satellite.id[4:].isdigit()


class TestSatelliteSwarm:
    """Tests for the SatelliteSwarm class"""

    def test_satellite_swarm_initialization(self, simulator_config):
        """Test that SatelliteSwarm initializes correctly"""
        swarm = SatelliteSwarm(simulator_config)
        assert len(swarm.satellites) == simulator_config.num_satellites
        assert swarm.stats["total_sent"] == 0
        assert swarm.stats["success"] == 0
        assert swarm.stats["errors"] == 0

    def test_correct_satellite_count(self, simulator_config):
        """Test that the correct number of satellites are created"""
        config = SimulatorConfig(
            num_satellites=50,
            points_per_second_per_satellite=10,
            api_url="http://localhost:8080",
            duration_seconds=1,
            max_connections=100,
            max_connections_per_host=50,
            anomaly_rate=0.01
        )
        swarm = SatelliteSwarm(config)
        assert len(swarm.satellites) == 50

    def test_satellite_ids_format(self, simulator_config):
        """Test that all satellite IDs follow the SAT-XXXX format"""
        swarm = SatelliteSwarm(simulator_config)
        for satellite in swarm.satellites:
            assert satellite.id.startswith("SAT-")
            # ID should be SAT- followed by 4 digits
            assert len(satellite.id) == 8
            assert satellite.id[4:].isdigit()

    def test_stats_initialization(self, simulator_config):
        """Test that statistics are properly initialized"""
        swarm = SatelliteSwarm(simulator_config)
        assert "total_sent" in swarm.stats
        assert "success" in swarm.stats
        assert "errors" in swarm.stats
        assert "start_time" in swarm.stats
        assert isinstance(swarm.stats["start_time"], float)

    def test_each_satellite_has_unique_generator(self, simulator_config):
        """Test that each satellite has its own generator instance"""
        swarm = SatelliteSwarm(simulator_config)
        generators = [sat.generator for sat in swarm.satellites]

        # All generators should be unique instances
        # (even if they have the same config)
        assert len(set(id(g) for g in generators)) == len(generators)

    def test_generators_have_correct_anomaly_rate(self, simulator_config):
        """Test that generators inherit the anomaly rate from config"""
        swarm = SatelliteSwarm(simulator_config)
        for satellite in swarm.satellites:
            assert satellite.generator.anomaly_rate == simulator_config.anomaly_rate


class TestSatelliteSwarmStatistics:
    """Tests for statistics tracking"""

    def test_stats_start_at_zero(self, simulator_config):
        """Test that all statistics start at zero"""
        swarm = SatelliteSwarm(simulator_config)
        assert swarm.stats["total_sent"] == 0
        assert swarm.stats["success"] == 0
        assert swarm.stats["errors"] == 0

    def test_stats_can_be_updated(self, simulator_config):
        """Test that statistics can be updated"""
        swarm = SatelliteSwarm(simulator_config)
        swarm.stats["total_sent"] = 100
        swarm.stats["success"] = 95
        swarm.stats["errors"] = 5

        assert swarm.stats["total_sent"] == 100
        assert swarm.stats["success"] == 95
        assert swarm.stats["errors"] == 5


class TestEdgeCases:
    """Tests for edge cases"""

    def test_single_satellite(self):
        """Test with just one satellite"""
        config = SimulatorConfig(
            num_satellites=1,
            points_per_second_per_satellite=10,
            api_url="http://localhost:8080",
            duration_seconds=1,
            max_connections=10,
            max_connections_per_host=5,
            anomaly_rate=0.01
        )
        swarm = SatelliteSwarm(config)
        assert len(swarm.satellites) == 1
        assert swarm.satellites[0].id == "SAT-0001"

    def test_large_satellite_count(self):
        """Test with a large number of satellites"""
        config = SimulatorConfig(
            num_satellites=1000,
            points_per_second_per_satellite=1,
            api_url="http://localhost:8080",
            duration_seconds=1,
            max_connections=1000,
            max_connections_per_host=500,
            anomaly_rate=0.01
        )
        swarm = SatelliteSwarm(config)
        assert len(swarm.satellites) == 1000

    def test_zero_anomaly_rate(self):
        """Test with zero anomaly rate"""
        config = SimulatorConfig(
            num_satellites=10,
            points_per_second_per_satellite=10,
            api_url="http://localhost:8080",
            duration_seconds=1,
            max_connections=10,
            max_connections_per_host=5,
            anomaly_rate=0.0
        )
        swarm = SatelliteSwarm(config)
        for satellite in swarm.satellites:
            assert satellite.generator.anomaly_rate == 0.0

    def test_high_anomaly_rate(self):
        """Test with 100% anomaly rate"""
        config = SimulatorConfig(
            num_satellites=10,
            points_per_second_per_satellite=10,
            api_url="http://localhost:8080",
            duration_seconds=1,
            max_connections=10,
            max_connections_per_host=5,
            anomaly_rate=1.0
        )
        swarm = SatelliteSwarm(config)
        for satellite in swarm.satellites:
            assert satellite.generator.anomaly_rate == 1.0

    def test_infinite_duration(self):
        """Test with zero duration (infinite run)"""
        config = SimulatorConfig(
            num_satellites=10,
            points_per_second_per_satellite=10,
            api_url="http://localhost:8080",
            duration_seconds=0,
            max_connections=10,
            max_connections_per_host=5,
            anomaly_rate=0.01
        )
        swarm = SatelliteSwarm(config)
        assert swarm.config.duration_seconds == 0
