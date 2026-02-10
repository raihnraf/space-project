"""
Tests for the TLE Manager class

Tests for downloading, caching, and accessing Two-Line Element (TLE) orbital data.
"""
import json
from datetime import datetime, timezone, timedelta
from pathlib import Path
from unittest.mock import Mock, patch, MagicMock

import pytest
import requests

from generators.tle_manager import TLEManager, TLEData, REAL_SATELLITES


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
def mock_celestrak_response():
    """Mock Celestrak TLE response"""
    return """ISS
1 25544U 98067A   24010.50000000  .00010000  00000-0  18000-3 0  9998
2 25544  51.6416 250.0000 0005000 200.0000 150.0000 15.50000000420000
NOAA-18
1 28654U 05018A   24010.50000000  .00000000  00000-0  00000-0 0  9998
2 28654  99.1000 200.0000 0014000 100.0000 260.0000 14.10000000450000
"""


@pytest.fixture
def temp_cache_dir(tmp_path):
    """Temporary cache directory for testing"""
    cache_dir = tmp_path / "cache"
    cache_dir.mkdir(parents=True, exist_ok=True)
    return cache_dir


@pytest.fixture
def tle_manager(temp_cache_dir):
    """Create a TLEManager with temporary cache directory"""
    return TLEManager(cache_dir=temp_cache_dir, cache_expiry_hours=24)


# =============================================================================
# Test TLEData
# =============================================================================


class TestTLEData:
    """Tests for TLEData dataclass"""

    def test_tle_data_creation(self):
        """Test creating TLEData object"""
        tle = TLEData(
            name="ISS",
            line1="1 25544U 98067A   24010.50000000  .00010000  00000-0  18000-3 0  9998",
            line2="2 25544  51.6416 250.0000 0005000 200.0000 150.0000 15.50000000420000"
        )
        assert tle.name == "ISS"
        assert tle.line1.startswith("1 ")
        assert tle.line2.startswith("2 ")

    def test_tle_data_string_representation(self):
        """Test string representation of TLEData"""
        tle = TLEData(
            name="TEST",
            line1="1 00000U 00000A   24010.50000000  .00000000  00000-0  00000-0 0  9999",
            line2="2 00000  00.0000 000.0000 0000000 000.0000 000.0000 00.00000000000000"
        )
        tle_str = str(tle)
        assert "TEST" in tle_str
        assert "1 00000U" in tle_str
        assert "2 00000" in tle_str


# =============================================================================
# Test TLEManager Initialization
# =============================================================================


class TestTLEManagerInitialization:
    """Tests for TLEManager initialization"""

    def test_initialization_with_custom_cache_dir(self, temp_cache_dir):
        """Test initialization with custom cache directory"""
        manager = TLEManager(cache_dir=temp_cache_dir, cache_expiry_hours=12)
        assert manager.cache_dir == temp_cache_dir
        assert manager.cache_expiry_hours == 12
        assert manager.cache_file == temp_cache_dir / "satellites.tle"
        assert manager.metadata_file == temp_cache_dir / "metadata.json"

    def test_initialization_creates_cache_dir(self, tmp_path):
        """Test that cache directory is created if it doesn't exist"""
        new_cache_dir = tmp_path / "new_cache"
        assert not new_cache_dir.exists()

        manager = TLEManager(cache_dir=new_cache_dir)
        assert new_cache_dir.exists()
        assert new_cache_dir.is_dir()

    def test_initialization_with_default_cache_dir(self):
        """Test initialization with default cache directory"""
        manager = TLEManager()
        assert manager.cache_dir == Path.home() / ".cache" / "orbitstream" / "tle"
        assert manager.cache_file.name == "satellites.tle"
        assert manager.metadata_file.name == "metadata.json"

    def test_initial_cache_is_empty(self, tle_manager):
        """Test that cache starts empty"""
        assert tle_manager._tle_cache == {}


# =============================================================================
# Test TLE Data Loading
# =============================================================================


class TestTLEDataLoading:
    """Tests for loading TLE data"""

    @patch('generators.tle_manager.requests.get')
    def test_load_tle_data_from_network(self, mock_get, tle_manager, mock_celestrak_response):
        """Test downloading TLE data from Celestrak"""
        # Mock the HTTP response
        mock_response = Mock()
        mock_response.text = mock_celestrak_response
        mock_response.raise_for_status = Mock()
        mock_get.return_value = mock_response

        # Load TLE data
        tle_data = tle_manager.load_tle_data()

        # Verify data was loaded
        assert len(tle_data) >= 2
        assert "ISS" in tle_data
        assert "NOAA-18" in tle_data

        # Verify ISS TLE structure
        iss_tle = tle_data["ISS"]
        assert iss_tle.name == "ISS"
        assert iss_tle.line1.startswith("1 25544")
        assert iss_tle.line2.startswith("2 25544")

    @patch('generators.tle_manager.requests.get')
    def test_load_tle_data_force_refresh(self, mock_get, tle_manager, mock_celestrak_response):
        """Test force_refresh parameter bypasses cache"""
        # Create existing cache
        tle_manager._tle_cache = {"OLD": TLEData("OLD", "1 line", "2 line")}
        tle_manager._save_to_cache()

        # Mock the HTTP response
        mock_response = Mock()
        mock_response.text = mock_celestrak_response
        mock_response.raise_for_status = Mock()
        mock_get.return_value = mock_response

        # Force refresh should download new data
        tle_data = tle_manager.load_tle_data(force_refresh=True)

        # Verify fresh data was loaded (not old cache)
        assert "ISS" in tle_data
        mock_get.assert_called_once()

    def test_load_tle_data_from_cache(self, tle_manager, sample_tle_data):
        """Test loading TLE data from cache"""
        # Populate cache
        tle_manager._tle_cache = sample_tle_data
        tle_manager._save_to_cache()

        # Clear in-memory cache
        tle_manager._tle_cache = {}

        # Load from cache (no network call)
        tle_data = tle_manager.load_tle_data()

        # Verify data was loaded from cache
        assert len(tle_data) == len(sample_tle_data)
        assert "ISS" in tle_data
        assert "NOAA-18" in tle_data

    def test_load_tle_data_cache_expiry(self, tle_manager, sample_tle_data):
        """Test that expired cache triggers refresh"""
        # Create old cache (expired)
        tle_manager._tle_cache = sample_tle_data
        tle_manager._save_to_cache()

        # Modify metadata to make cache appear old
        old_time = datetime.now(timezone.utc) - timedelta(hours=25)
        with open(tle_manager.metadata_file, 'w') as f:
            json.dump({'cached_at': old_time.isoformat()}, f)

        # Clear in-memory cache
        tle_manager._tle_cache = {}

        # With mock for network (using fallback if network fails)
        with patch('generators.tle_manager.requests.get') as mock_get:
            mock_get.side_effect = requests.RequestException("Network error")
            tle_data = tle_manager.load_tle_data()

            # Should have fallen back to minimal TLE data
            assert len(tle_data) >= 1


# =============================================================================
# Test Satellite TLE Retrieval
# =============================================================================


class TestGetSatelliteTLE:
    """Tests for get_satellite_tle method"""

    def test_get_satellite_tle_exact_match(self, tle_manager, sample_tle_data):
        """Test getting TLE with exact name match"""
        tle_manager._tle_cache = sample_tle_data

        tle = tle_manager.get_satellite_tle("ISS")
        assert tle is not None
        assert tle.name == "ISS"

    def test_get_satellite_tle_case_insensitive(self, tle_manager, sample_tle_data):
        """Test case-insensitive satellite name matching"""
        tle_manager._tle_cache = sample_tle_data

        # Test various case combinations
        for name in ["iss", "Iss", "ISS", "iSs"]:
            tle = tle_manager.get_satellite_tle(name)
            assert tle is not None
            assert tle.name == "ISS"

    def test_get_satellite_tle_partial_match(self, tle_manager, sample_tle_data):
        """Test partial name matching"""
        tle_manager._tle_cache = sample_tle_data

        tle = tle_manager.get_satellite_tle("STARLINK")
        assert tle is not None
        assert "STARLINK" in tle.name.upper()

    def test_get_satellite_tle_not_found(self, tle_manager, sample_tle_data):
        """Test getting TLE for non-existent satellite"""
        tle_manager._tle_cache = sample_tle_data

        tle = tle_manager.get_satellite_tle("NONEXISTENT")
        assert tle is None


# =============================================================================
# Test Available Satellites
# =============================================================================


class TestAvailableSatellites:
    """Tests for available satellites list"""

    def test_get_available_satellites(self, tle_manager, sample_tle_data):
        """Test getting list of available satellites"""
        tle_manager._tle_cache = sample_tle_data

        satellites = tle_manager.get_available_satellites()
        assert len(satellites) == len(sample_tle_data)
        assert "ISS" in satellites
        assert "NOAA-18" in satellites

    def test_get_real_satellite_names(self, tle_manager, sample_tle_data):
        """Test getting real satellite names"""
        tle_manager._tle_cache = sample_tle_data

        satellites = tle_manager.get_real_satellite_names(count=3)
        assert len(satellites) <= 3

        # All returned satellites should be in our cache
        for name in satellites:
            assert tle_manager.get_satellite_tle(name) is not None

    def test_get_real_satellite_names_more_than_available(self, tle_manager):
        """Test requesting more satellites than available"""
        # Only add ISS to cache
        tle_manager._tle_cache = {
            "ISS": TLEData("ISS", "1 line", "2 line")
        }

        satellites = tle_manager.get_real_satellite_names(count=10)
        # Should cycle through available satellites
        assert len(satellites) == 10
        assert all(name == "ISS" for name in satellites)


# =============================================================================
# Test Cache Management
# =============================================================================


class TestCacheManagement:
    """Tests for cache file management"""

    def test_save_to_cache(self, tle_manager, sample_tle_data):
        """Test saving TLE data to cache file"""
        tle_manager._tle_cache = sample_tle_data
        tle_manager._save_to_cache()

        # Verify cache file exists
        assert tle_manager.cache_file.exists()

        # Verify metadata file exists
        assert tle_manager.metadata_file.exists()

        # Verify metadata content
        with open(tle_manager.metadata_file, 'r') as f:
            metadata = json.load(f)

        assert 'cached_at' in metadata
        assert 'satellite_count' in metadata
        assert metadata['satellite_count'] == len(sample_tle_data)

    def test_load_from_cache(self, tle_manager, sample_tle_data):
        """Test loading TLE data from cache file"""
        # Save data first
        tle_manager._tle_cache = sample_tle_data
        tle_manager._save_to_cache()

        # Clear in-memory cache
        tle_manager._tle_cache = {}

        # Load from cache
        tle_manager._load_from_cache()

        # Verify data was loaded
        assert len(tle_manager._tle_cache) == len(sample_tle_data)
        assert "ISS" in tle_manager._tle_cache
        assert tle_manager._tle_cache["ISS"].name == "ISS"

    def test_is_cache_valid_with_valid_cache(self, tle_manager):
        """Test cache validity check with valid cache"""
        # Create valid cache
        tle_manager._save_to_cache()

        # Cache should be valid immediately after creation
        assert tle_manager._is_cache_valid() is True

    def test_is_cache_valid_with_expired_cache(self, tle_manager):
        """Test cache validity check with expired cache"""
        # Create cache
        tle_manager._save_to_cache()

        # Modify metadata to make cache appear old
        old_time = datetime.now(timezone.utc) - timedelta(hours=25)
        with open(tle_manager.metadata_file, 'w') as f:
            json.dump({'cached_at': old_time.isoformat()}, f)

        # Cache should be invalid
        assert tle_manager._is_cache_valid() is False

    def test_is_cache_valid_without_cache_file(self, tle_manager):
        """Test cache validity check without cache file"""
        assert tle_manager._is_cache_valid() is False

    def test_is_cache_valid_without_metadata_file(self, tle_manager):
        """Test cache validity check without metadata file"""
        # Create cache file but not metadata
        tle_manager.cache_file.touch()

        assert tle_manager._is_cache_valid() is False


# =============================================================================
# Test Fallback Behavior
# =============================================================================


class TestFallbackBehavior:
    """Tests for fallback TLE data when network fails"""

    @patch('generators.tle_manager.requests.get')
    def test_fallback_tle_on_network_failure(self, mock_get, tle_manager):
        """Test that fallback TLE is used when network fails"""
        # Mock network failure
        mock_get.side_effect = requests.RequestException("Network error")

        # Load TLE data (should use fallback)
        tle_data = tle_manager.load_tle_data()

        # Should have at least ISS from fallback
        assert "ISS" in tle_data

    @patch('generators.tle_manager.requests.get')
    def test_fallback_tle_structure(self, mock_get, tle_manager):
        """Test that fallback TLE has correct structure"""
        mock_get.side_effect = requests.RequestException("Network error")

        tle_data = tle_manager.load_tle_data()

        # Verify ISS fallback TLE
        iss_tle = tle_data["ISS"]
        assert iss_tle.name == "ISS"
        assert iss_tle.line1.startswith("1 25544")
        assert iss_tle.line2.startswith("2 25544")


# =============================================================================
# Test Network Download
# =============================================================================


class TestNetworkDownload:
    """Tests for TLE data download from network"""

    @patch('generators.tle_manager.requests.get')
    def test_download_tle_data_success(self, mock_get, tle_manager, mock_celestrak_response):
        """Test successful TLE data download"""
        mock_response = Mock()
        mock_response.text = mock_celestrak_response
        mock_response.raise_for_status = Mock()
        mock_get.return_value = mock_response

        tle_manager._download_tle_data()

        # Verify data was loaded
        assert len(tle_manager._tle_cache) >= 2
        assert "ISS" in tle_manager._tle_cache
        assert "NOAA-18" in tle_manager._tle_cache

    @patch('generators.tle_manager.requests.get')
    def test_download_tle_data_request_exception(self, mock_get, tle_manager):
        """Test handling of network request exception"""
        mock_get.side_effect = requests.RequestException("Network error")

        # Should not raise exception, should use fallback
        tle_manager._download_tle_data()

        # Should have fallback data
        assert len(tle_manager._tle_cache) >= 1

    @patch('generators.tle_manager.requests.get')
    def test_download_tle_data_http_error(self, mock_get, tle_manager):
        """Test handling of HTTP error"""
        mock_get.side_effect = requests.HTTPError("404 Not Found")

        # Should not raise exception, should use fallback
        tle_manager._download_tle_data()

        # Should have fallback data
        assert len(tle_manager._tle_cache) >= 1

    @patch('generators.tle_manager.requests.get')
    def test_download_tle_data_timeout(self, mock_get, tle_manager):
        """Test handling of network timeout"""
        mock_get.side_effect = requests.Timeout("Connection timeout")

        # Should not raise exception, should use fallback
        tle_manager._download_tle_data()

        # Should have fallback data
        assert len(tle_manager._tle_cache) >= 1


# =============================================================================
# Test Real Satellites List
# =============================================================================


class TestRealSatellitesList:
    """Tests for REAL_SATELLITES constant"""

    def test_real_satellites_list_not_empty(self):
        """Test that REAL_SATELLITES list is not empty"""
        assert len(REAL_SATELLITES) > 0

    def test_real_satellites_list_contains_iss(self):
        """Test that ISS is in the list"""
        assert "ISS" in REAL_SATELLITES

    def test_real_satellites_list_contains_starlink(self):
        """Test that Starlink satellites are in the list"""
        starlink_count = sum(1 for name in REAL_SATELLITES if "STARLINK" in name.upper())
        assert starlink_count > 0

    def test_real_satellites_list_contains_noaa(self):
        """Test that NOAA satellites are in the list"""
        noaa_count = sum(1 for name in REAL_SATELLITES if "NOAA" in name.upper())
        assert noaa_count > 0


# =============================================================================
# Integration Tests
# =============================================================================


class TestIntegration:
    """Integration tests for TLEManager"""

    def test_full_cache_workflow(self, tle_manager, sample_tle_data):
        """Test complete cache workflow: save -> load -> retrieve"""
        # Save data
        tle_manager._tle_cache = sample_tle_data
        tle_manager._save_to_cache()

        # Clear and reload
        tle_manager._tle_cache = {}
        tle_manager._load_from_cache()

        # Retrieve data
        iss_tle = tle_manager.get_satellite_tle("ISS")

        assert iss_tle is not None
        assert iss_tle.name == "ISS"
        assert iss_tle.line1 == sample_tle_data["ISS"].line1
        assert iss_tle.line2 == sample_tle_data["ISS"].line2

    def test_multiple_tle_managers_independent(self, temp_cache_dir):
        """Test that multiple TLEManager instances are independent"""
        manager1 = TLEManager(cache_dir=temp_cache_dir / "manager1")
        manager2 = TLEManager(cache_dir=temp_cache_dir / "manager2")

        # Add different data to each
        manager1._tle_cache = {"SAT1": TLEData("SAT1", "1 line1", "2 line1")}
        manager2._tle_cache = {"SAT2": TLEData("SAT2", "1 line2", "2 line2")}

        # Verify independence
        assert "SAT1" in manager1._tle_cache
        assert "SAT2" in manager2._tle_cache
        assert "SAT2" not in manager1._tle_cache
        assert "SAT1" not in manager2._tle_cache
