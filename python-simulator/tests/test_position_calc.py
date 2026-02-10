"""
Tests for the Position Calculator class

Tests for satellite position calculation using TLE data and Skyfield library.
"""
from datetime import datetime, timezone, timedelta
from unittest.mock import Mock, patch, MagicMock

import pytest
from skyfield.api import EarthSatellite

from generators.position_calc import PositionCalculator, PositionData, create_position_manager
from generators.tle_manager import TLEData


# =============================================================================
# Test Fixtures
# =============================================================================


@pytest.fixture
def sample_tle_data():
    """Sample TLE data for testing"""
    return {
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
        "STARLINK-1001": TLEData(
            "STARLINK-1001",
            "1 44713U 19062A   24010.50000000  .00000000  00000-0  00000-0 0  9998",
            "2 44713  53.0000 100.0000 0001000 100.0000 260.0000 15.00000000500000"
        ),
    }


@pytest.fixture
def position_calculator(sample_tle_data):
    """Create a PositionCalculator with sample TLE data"""
    return PositionCalculator(sample_tle_data)


@pytest.fixture
def sample_timestamp():
    """Sample timestamp for testing"""
    return datetime(2024, 2, 10, 15, 30, 0, tzinfo=timezone.utc)


# =============================================================================
# Test PositionData
# =============================================================================


class TestPositionData:
    """Tests for PositionData dataclass"""

    def test_position_data_creation(self):
        """Test creating PositionData object"""
        pos = PositionData(
            latitude=40.7128,
            longitude=-74.0060,
            altitude_km=408.5,
            velocity_kmph=27576.5
        )
        assert pos.latitude == 40.7128
        assert pos.longitude == -74.0060
        assert pos.altitude_km == 408.5
        assert pos.velocity_kmph == 27576.5

    def test_position_data_repr(self):
        """Test string representation of PositionData"""
        pos = PositionData(
            latitude=40.7128,
            longitude=-74.0060,
            altitude_km=408.5,
            velocity_kmph=27576.5
        )
        repr_str = repr(pos)
        assert "40.7128" in repr_str
        assert "-74.0060" in repr_str
        assert "408.50" in repr_str
        assert "27576.50" in repr_str


# =============================================================================
# Test PositionCalculator Initialization
# =============================================================================


class TestPositionCalculatorInitialization:
    """Tests for PositionCalculator initialization"""

    def test_initialization_with_tle_data(self, sample_tle_data):
        """Test initialization with TLE data"""
        calc = PositionCalculator(sample_tle_data)
        assert calc.tle_data == sample_tle_data
        assert isinstance(calc.satellites, dict)

    def test_loads_satellites_from_tle(self, position_calculator):
        """Test that satellites are loaded from TLE data"""
        # Should have loaded satellites from TLE data
        assert len(position_calculator.satellites) > 0
        assert "ISS" in position_calculator.satellites

    def test_earth_radius_constant(self):
        """Test Earth radius constant"""
        assert PositionCalculator.EARTH_RADIUS_KM == 6371.0

    def test_timescale_loaded(self, position_calculator):
        """Test that Skyfield timescale is loaded"""
        assert position_calculator.ts is not None


# =============================================================================
# Test Position Calculation
# =============================================================================


class TestGetPosition:
    """Tests for get_position method"""

    def test_get_position_returns_position_data(self, position_calculator, sample_timestamp):
        """Test that get_position returns PositionData"""
        pos = position_calculator.get_position("ISS", sample_timestamp)
        assert pos is not None
        assert isinstance(pos, PositionData)

    def test_get_position_lat_lon_ranges(self, position_calculator, sample_timestamp):
        """Test that latitude and longitude are in valid ranges"""
        pos = position_calculator.get_position("ISS", sample_timestamp)
        assert -90 <= pos.latitude <= 90
        assert -180 <= pos.longitude <= 180

    def test_get_position_altitude_positive(self, position_calculator, sample_timestamp):
        """Test that altitude is positive for LEO satellites"""
        pos = position_calculator.get_position("ISS", sample_timestamp)
        assert pos.altitude_km > 0
        # LEO altitude range: ~300-2000 km
        assert 300 < pos.altitude_km < 2000

    def test_get_position_velocity_reasonable(self, position_calculator, sample_timestamp):
        """Test that velocity is in orbital range (~27,000 km/h for LEO)"""
        pos = position_calculator.get_position("ISS", sample_timestamp)
        # Orbital velocity should be approximately 25,000-30,000 km/h
        assert 25000 < pos.velocity_kmph < 30000

    def test_get_position_case_insensitive(self, position_calculator, sample_timestamp):
        """Test case-insensitive satellite name matching"""
        pos1 = position_calculator.get_position("ISS", sample_timestamp)
        pos2 = position_calculator.get_position("iss", sample_timestamp)
        pos3 = position_calculator.get_position("IsS", sample_timestamp)

        # All should return valid positions
        assert pos1 is not None
        assert pos2 is not None
        assert pos3 is not None

    def test_get_position_partial_match(self, position_calculator, sample_timestamp):
        """Test partial satellite name matching"""
        pos = position_calculator.get_position("NOAA", sample_timestamp)
        assert pos is not None

    def test_get_position_unknown_satellite(self, position_calculator, sample_timestamp):
        """Test with unknown satellite name"""
        pos = position_calculator.get_position("UNKNOWN-SAT", sample_timestamp)
        assert pos is None

    def test_get_position_different_satellites(self, position_calculator, sample_timestamp):
        """Test that different satellites return different positions"""
        pos_iss = position_calculator.get_position("ISS", sample_timestamp)
        pos_noaa = position_calculator.get_position("NOAA-18", sample_timestamp)

        # Positions should be different (same time, different orbits)
        assert pos_iss.latitude != pos_noaa.latitude or pos_iss.longitude != pos_noaa.longitude

    def test_get_position_naive_timestamp(self, position_calculator):
        """Test with naive datetime (no timezone)"""
        naive_time = datetime(2024, 2, 10, 15, 30, 0)  # No timezone
        pos = position_calculator.get_position("ISS", naive_time)
        assert pos is not None
        # Should treat naive time as UTC

    def test_get_position_non_utc_timezone(self, position_calculator):
        """Test with non-UTC timezone"""
        # Create timezone that's UTC+8
        from datetime import timezone, timedelta
        utc_plus_8 = timezone(timedelta(hours=8))
        local_time = datetime(2024, 2, 10, 15, 30, 0, tzinfo=utc_plus_8)

        pos = position_calculator.get_position("ISS", local_time)
        assert pos is not None
        # Position should be calculated for the equivalent UTC time

    def test_get_position_changes_with_time(self, position_calculator):
        """Test that position changes over time"""
        t1 = datetime(2024, 2, 10, 12, 0, 0, tzinfo=timezone.utc)
        t2 = datetime(2024, 2, 10, 13, 0, 0, tzinfo=timezone.utc)

        pos1 = position_calculator.get_position("ISS", t1)
        pos2 = position_calculator.get_position("ISS", t2)

        # Positions should be different (satellite moved)
        # At least one coordinate should be different
        assert pos1.latitude != pos2.latitude or pos1.longitude != pos2.longitude


# =============================================================================
# Test Velocity Calculation
# =============================================================================


class TestGetVelocity:
    """Tests for get_velocity method"""

    def test_get_velocity_returns_float(self, position_calculator, sample_timestamp):
        """Test that get_velocity returns a float"""
        velocity = position_calculator.get_velocity("ISS", sample_timestamp)
        assert isinstance(velocity, float)

    def test_get_velocity_reasonable_range(self, position_calculator, sample_timestamp):
        """Test that velocity is in expected orbital range"""
        velocity = position_calculator.get_velocity("ISS", sample_timestamp)
        # LEO orbital velocity: ~25,000-30,000 km/h
        assert 25000 < velocity < 30000

    def test_get_velocity_unknown_satellite(self, position_calculator, sample_timestamp):
        """Test get_velocity with unknown satellite"""
        velocity = position_calculator.get_velocity("UNKNOWN", sample_timestamp)
        assert velocity is None


# =============================================================================
# Test Satellite Visibility
# =============================================================================


class TestIsSatelliteVisible:
    """Tests for is_satellite_visible method"""

    def test_is_satellite_visible_returns_bool(self, position_calculator, sample_timestamp):
        """Test that is_satellite_visible returns a boolean"""
        visible = position_calculator.is_satellite_visible(
            "ISS", sample_timestamp,
            observer_lat=40.7128,  # New York City
            observer_lon=-74.0060
        )
        assert isinstance(visible, bool)

    def test_is_satellite_visible_unknown_satellite(self, position_calculator, sample_timestamp):
        """Test is_satellite_visible with unknown satellite"""
        visible = position_calculator.is_satellite_visible(
            "UNKNOWN", sample_timestamp,
            observer_lat=0.0,
            observer_lon=0.0
        )
        assert visible is None

    def test_is_satellite_visible_with_min_elevation(self, position_calculator, sample_timestamp):
        """Test with different minimum elevation angles"""
        # Lower elevation angle should be more likely to be visible
        visible_low = position_calculator.is_satellite_visible(
            "ISS", sample_timestamp,
            observer_lat=0.0,
            observer_lon=0.0,
            min_elevation_deg=0.0
        )

        visible_high = position_calculator.is_satellite_visible(
            "ISS", sample_timestamp,
            observer_lat=0.0,
            observer_lon=0.0,
            min_elevation_deg=45.0
        )

        # Both should return boolean values
        assert isinstance(visible_low, bool)
        assert isinstance(visible_high, bool)


# =============================================================================
# Test Utility Methods
# =============================================================================


class TestUtilityMethods:
    """Tests for utility methods"""

    def test_get_available_satellites(self, position_calculator):
        """Test getting list of available satellites"""
        satellites = position_calculator.get_available_satellites()
        assert isinstance(satellites, list)
        assert len(satellites) > 0
        assert "ISS" in satellites

    def test_get_orbital_period(self, position_calculator):
        """Test getting orbital period"""
        period = position_calculator.get_orbital_period("ISS")
        assert period is not None
        # ISS orbital period is approximately 90-95 minutes
        assert 85 < period < 100

    def test_get_orbital_period_unknown_satellite(self, position_calculator):
        """Test get_orbital_period with unknown satellite"""
        period = position_calculator.get_orbital_period("UNKNOWN")
        assert period is None


# =============================================================================
# Test Position Value Ranges
# =============================================================================


class TestPositionValueRanges:
    """Tests for position value ranges over time"""

    def test_latitude_stays_in_range(self, position_calculator):
        """Test that latitude stays within valid range over multiple time points"""
        start = datetime(2024, 2, 10, 0, 0, 0, tzinfo=timezone.utc)
        for i in range(24):  # Check every hour for 24 hours
            t = start + timedelta(hours=i)
            pos = position_calculator.get_position("ISS", t)
            assert -90 <= pos.latitude <= 90, \
                f"Latitude {pos.latitude} out of range at hour {i}"

    def test_longitude_normalization(self, position_calculator):
        """Test that longitude is normalized to -180 to 180 range"""
        start = datetime(2024, 2, 10, 0, 0, 0, tzinfo=timezone.utc)
        for i in range(24):
            t = start + timedelta(hours=i)
            pos = position_calculator.get_position("ISS", t)
            assert -180 <= pos.longitude <= 180, \
                f"Longitude {pos.longitude} out of range at hour {i}"

    def test_altitude_stays_positive(self, position_calculator):
        """Test that altitude stays positive"""
        start = datetime(2024, 2, 10, 0, 0, 0, tzinfo=timezone.utc)
        for i in range(24):
            t = start + timedelta(hours=i)
            pos = position_calculator.get_position("ISS", t)
            assert pos.altitude_km > 0, \
                f"Altitude {pos.altitude_km} is not positive at hour {i}"

    def test_velocity_stays_consistent(self, position_calculator):
        """Test that velocity stays consistent (orbital mechanics)"""
        start = datetime(2024, 2, 10, 0, 0, 0, tzinfo=timezone.utc)
        velocities = []
        for i in range(24):
            t = start + timedelta(hours=i)
            pos = position_calculator.get_position("ISS", t)
            velocities.append(pos.velocity_kmph)

        # All velocities should be in orbital range
        for v in velocities:
            assert 25000 < v < 30000, \
                f"Velocity {v} out of orbital range"

        # Velocity shouldn't vary much for circular LEO orbit
        velocity_variation = max(velocities) - min(velocities)
        assert velocity_variation < 5000, \
            f"Velocity varies too much: {velocity_variation}"


# =============================================================================
# Test Different Satellites
# =============================================================================


class TestDifferentSatellites:
    """Tests with different satellite types"""

    def test_iss_position_calculation(self, position_calculator, sample_timestamp):
        """Test ISS position calculation"""
        pos = position_calculator.get_position("ISS", sample_timestamp)
        assert pos is not None
        # ISS altitude is approximately 408 km
        assert 350 < pos.altitude_km < 500
        # ISS inclination is 51.6 degrees, so it covers most of Earth
        assert -90 <= pos.latitude <= 90

    def test_noaa_position_calculation(self, position_calculator, sample_timestamp):
        """Test NOAA satellite position calculation"""
        pos = position_calculator.get_position("NOAA-18", sample_timestamp)
        assert pos is not None
        # NOAA satellites are in polar orbits (~850 km)
        assert 700 < pos.altitude_km < 1000

    def test_starlink_position_calculation(self, position_calculator, sample_timestamp):
        """Test Starlink satellite position calculation"""
        pos = position_calculator.get_position("STARLINK-1001", sample_timestamp)
        assert pos is not None
        # Starlink satellites are in lower LEO (~550 km)
        assert 400 < pos.altitude_km < 700


# =============================================================================
# Integration Tests
# =============================================================================


class TestIntegration:
    """Integration tests for position calculation"""

    def test_position_tracking_over_orbit(self, position_calculator):
        """Test tracking satellite position over one orbit"""
        # ISS completes an orbit in ~90 minutes
        start = datetime(2024, 2, 10, 0, 0, 0, tzinfo=timezone.utc)
        positions = []

        for i in range(0, 100, 5):  # Every 5 minutes for 100 minutes
            t = start + timedelta(minutes=i)
            pos = position_calculator.get_position("ISS", t)
            positions.append(pos)

        # Should have collected multiple positions
        assert len(positions) > 15

        # Verify satellite moved (positions are different)
        unique_positions = set((p.latitude, p.longitude) for p in positions)
        assert len(unique_positions) > len(positions) / 2, \
            "Satellite should have moved significantly"

    def test_position_data_consistency(self, position_calculator, sample_timestamp):
        """Test that position data is internally consistent"""
        pos = position_calculator.get_position("ISS", sample_timestamp)

        # Altitude should be positive and realistic
        assert pos.altitude_km > 0
        assert pos.altitude_km < 10000

        # Velocity should be consistent with altitude (lower = faster)
        # For circular orbit: v = sqrt(GM/r)
        # At 400km: ~27600 km/h
        assert 25000 < pos.velocity_kmph < 30000

    @patch('generators.position_calc.TLEManager')
    def test_create_position_manager(self, mock_tle_manager_class):
        """Test create_position_manager convenience function"""
        # Mock TLE manager
        mock_tle_manager = Mock()
        mock_tle_data = {
            "TEST": TLEData("TEST", "1 line", "2 line")
        }
        mock_tle_manager.load_tle_data.return_value = mock_tle_data
        mock_tle_manager_class.return_value = mock_tle_manager

        # Create position manager
        calc = create_position_manager()

        # Verify it was created correctly
        assert isinstance(calc, PositionCalculator)
        mock_tle_manager_class.assert_called_once()
        mock_tle_manager.load_tle_data.assert_called_once()
