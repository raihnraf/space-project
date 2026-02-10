"""
TLE Manager for OrbitStream Satellite Simulator

Downloads and caches Two-Line Element (TLE) orbital data from Celestrak.
Provides real satellite orbital data for position calculations.
"""

import hashlib
import json
import logging
import os
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path
from typing import Dict, Optional

import requests

logger = logging.getLogger(__name__)

# Celestrak TLE data URLs
CELESTRAK_BASE_URL = "https://celestrak.org/NORAD/elements/gp.php"
CELESTRAK_ACTIVE_URL = f"{CELESTRAK_BASE_URL}?GROUP=active&FORMAT=tle"

# Real satellites to use (ISS, Starlink, GPS, etc.)
# Using a diverse set of LEO satellites for realistic simulation
REAL_SATELLITES = [
    # ISS (International Space Station)
    "ISS",
    # Starlink satellites (small sample)
    "STARLINK-1001",
    "STARLINK-1002",
    "STARLINK-1003",
    "STARLINK-1004",
    "STARLINK-1005",
    # NOAA weather satellites
    "NOAA-18",
    "NOAA-19",
    "METOP-B",
    "METOP-C",
    # CubeSats and small satellites
    "FUNCUBE-1",
    "LILACSAT-2",
    "BUGSAT-1",
    # More Starlink
    "STARLINK-1006",
    "STARLINK-1007",
    "STARLINK-1008",
    "STARLINK-1009",
    "STARLINK-1010",
    # NOAA
    "NOAA-20",
    # More Starlink
    "STARLINK-1011",
    "STARLINK-1012",
    "STARLINK-1013",
    "STARLINK-1014",
    "STARLINK-1015",
    # CubeSats
    "AAUSAT-4",
    "OUTSAT-1",
    # More Starlink
    "STARLINK-1016",
    "STARLINK-1017",
    "STARLINK-1018",
    "STARLINK-1019",
    "STARLINK-1020",
    # Additional satellites
    "CAPE-2",
    "CUBESAT-H",
    "CHALLENGER",
    # More Starlink to fill the list
    "STARLINK-1021",
    "STARLINK-1022",
    "STARLINK-1023",
    "STARLINK-1024",
    "STARLINK-1025",
    "STARLINK-1026",
    "STARLINK-1027",
    "STARLINK-1028",
    "STARLINK-1029",
    "STARLINK-1030",
    "STARLINK-1031",
    "STARLINK-1032",
    "STARLINK-1033",
    "STARLINK-1034",
    "STARLINK-1035",
    "STARLINK-1036",
    "STARLINK-1037",
    "STARLINK-1038",
    "STARLINK-1039",
    "STARLINK-1040",
    "STARLINK-1041",
    "STARLINK-1042",
    "STARLINK-1043",
    "STARLINK-1044",
    "STARLINK-1045",
    "STARLINK-1046",
    "STARLINK-1047",
    "STARLINK-1048",
    "STARLINK-1049",
    "STARLINK-1050",
    # Additional diverse satellites
    "QO-100",  # Es'hail-2 (geostationary amateur radio satellite)
    "AO-91",   # Fox-1B
    "AO-92",   # Fox-1D
    "MO-112",  # MEMESAT-1
    "TO-108",  # Tubesat-1
    # More Starlink
    "STARLINK-1051",
    "STARLINK-1052",
    "STARLINK-1053",
    "STARLINK-1054",
    "STARLINK-1055",
    "STARLINK-1056",
    "STARLINK-1057",
    "STARLINK-1058",
    "STARLINK-1059",
    "STARLINK-1060",
    "STARLINK-1061",
    "STARLINK-1062",
    "STARLINK-1063",
    "STARLINK-1064",
    "STARLINK-1065",
    "STARLINK-1066",
    "STARLINK-1067",
    "STARLINK-1068",
    "STARLINK-1069",
    "STARLINK-1070",
    # Weather and research satellites
    "FENGYUN-3C",
    "METEOR-M2",
    "METEOR-M2-2",
    # More Starlink
    "STARLINK-1071",
    "STARLINK-1072",
    "STARLINK-1073",
    "STARLINK-1074",
    "STARLINK-1075",
    "STARLINK-1076",
    "STARLINK-1077",
    "STARLINK-1078",
    "STARLINK-1079",
    "STARLINK-1080",
]


@dataclass
class TLEData:
    """Two-Line Element orbital data for a satellite"""
    name: str
    line1: str
    line2: str

    def __str__(self) -> str:
        return f"{self.name}\n{self.line1}\n{self.line2}"


class TLEManager:
    """
    Manages TLE (Two-Line Element) orbital data for satellites.

    Downloads TLE data from Celestrak, caches locally for faster startup,
    and provides access to satellite orbital elements.
    """

    def __init__(self, cache_dir: Optional[Path] = None, cache_expiry_hours: int = 24):
        """
        Initialize TLE Manager.

        Args:
            cache_dir: Directory to cache TLE data (default: ~/.cache/orbitstream/tle)
            cache_expiry_hours: How long to cache TLE data before refreshing (default: 24 hours)
        """
        if cache_dir is None:
            cache_dir = Path.home() / ".cache" / "orbitstream" / "tle"

        self.cache_dir = Path(cache_dir)
        self.cache_dir.mkdir(parents=True, exist_ok=True)

        self.cache_file = self.cache_dir / "satellites.tle"
        self.metadata_file = self.cache_dir / "metadata.json"
        self.cache_expiry_hours = cache_expiry_hours

        self._tle_cache: Dict[str, TLEData] = {}

    def load_tle_data(self, force_refresh: bool = False) -> Dict[str, TLEData]:
        """
        Load TLE data from cache or download fresh data.

        Args:
            force_refresh: Force downloading fresh data even if cache is valid

        Returns:
            Dictionary mapping satellite names to TLEData objects
        """
        # Try to load from cache first
        if not force_refresh and self._is_cache_valid():
            logger.info("Loading TLE data from cache...")
            self._load_from_cache()
            if self._tle_cache:
                logger.info(f"Loaded {len(self._tle_cache)} satellites from cache")
                return self._tle_cache

        # Download fresh data
        logger.info("Downloading TLE data from Celestrak...")
        self._download_tle_data()
        self._save_to_cache()
        logger.info(f"Downloaded {len(self._tle_cache)} satellites")

        return self._tle_cache

    def get_satellite_tle(self, satellite_name: str) -> Optional[TLEData]:
        """
        Get TLE data for a specific satellite.

        Args:
            satellite_name: Name of the satellite (case-insensitive)

        Returns:
            TLEData object or None if not found
        """
        # Try exact match (case-insensitive)
        for name, tle in self._tle_cache.items():
            if name.upper() == satellite_name.upper():
                return tle

        # Try partial match
        for name, tle in self._tle_cache.items():
            if satellite_name.upper() in name.upper():
                return tle

        return None

    def get_available_satellites(self) -> list[str]:
        """Get list of available satellite names in the cache"""
        return list(self._tle_cache.keys())

    def get_real_satellite_names(self, count: int = 100) -> list[str]:
        """
        Get a list of real satellite names for simulation.

        Returns the requested number of satellite names from our predefined list.
        If we have more satellites available than requested, returns a subset.

        Args:
            count: Number of satellite names to return

        Returns:
            List of satellite names
        """
        # Ensure we have TLE data loaded
        if not self._tle_cache:
            self.load_tle_data()

        # Filter to only satellites we have TLE data for
        available = []
        for name in REAL_SATELLITES[:count]:
            if self.get_satellite_tle(name):
                available.append(name)

        # If we don't have enough, cycle through what we do have
        while len(available) < count:
            for name in REAL_SATELLITES:
                if len(available) >= count:
                    break
                if self.get_satellite_tle(name) and name not in available:
                    available.append(name)

        return available[:count]

    def _is_cache_valid(self) -> bool:
        """Check if cached TLE data is still valid"""
        if not self.cache_file.exists():
            return False

        if not self.metadata_file.exists():
            return False

        try:
            with open(self.metadata_file, 'r') as f:
                metadata = json.load(f)

            cached_time = datetime.fromisoformat(metadata.get('cached_at', ''))
            age = datetime.now(timezone.utc) - cached_time

            return age.total_seconds() < (self.cache_expiry_hours * 3600)
        except (json.JSONDecodeError, ValueError, KeyError):
            return False

    def _load_from_cache(self) -> None:
        """Load TLE data from cache file"""
        self._tle_cache.clear()

        try:
            with open(self.cache_file, 'r') as f:
                content = f.read()

            # Parse TLE format (3 lines per satellite: name, line1, line2)
            lines = content.strip().split('\n')
            i = 0
            while i < len(lines):
                name = lines[i].strip()
                if i + 2 < len(lines):
                    line1 = lines[i + 1].strip()
                    line2 = lines[i + 2].strip()
                    self._tle_cache[name] = TLEData(name, line1, line2)
                i += 3

        except (IOError, IndexError) as e:
            logger.warning(f"Failed to load TLE cache: {e}")
            self._tle_cache.clear()

    def _save_to_cache(self) -> None:
        """Save TLE data to cache file"""
        try:
            # Save TLE data
            with open(self.cache_file, 'w') as f:
                for tle in self._tle_cache.values():
                    f.write(str(tle))
                    f.write('\n')

            # Save metadata
            metadata = {
                'cached_at': datetime.now(timezone.utc).isoformat(),
                'satellite_count': len(self._tle_cache),
                'source': CELESTRAK_ACTIVE_URL
            }
            with open(self.metadata_file, 'w') as f:
                json.dump(metadata, f, indent=2)

            logger.info(f"Saved {len(self._tle_cache)} satellites to cache")

        except IOError as e:
            logger.warning(f"Failed to save TLE cache: {e}")

    def _download_tle_data(self) -> None:
        """Download TLE data from Celestrak"""
        self._tle_cache.clear()

        try:
            response = requests.get(CELESTRAK_ACTIVE_URL, timeout=30)
            response.raise_for_status()

            content = response.text
            lines = content.strip().split('\n')

            # Parse TLE format
            i = 0
            while i < len(lines):
                line = lines[i].strip()

                # Skip empty lines and comments
                if not line or line.startswith('#'):
                    i += 1
                    continue

                # Check if this is a satellite name line
                # (doesn't start with '1 ' or '2 ')
                if not line.startswith('1 ') and not line.startswith('2 '):
                    name = line
                    if i + 2 < len(lines):
                        line1 = lines[i + 1].strip()
                        line2 = lines[i + 2].strip()

                        # Validate TLE format
                        if line1.startswith('1 ') and line2.startswith('2 '):
                            self._tle_cache[name] = TLEData(name, line1, line2)
                            i += 3
                        else:
                            i += 1
                    else:
                        i += 1
                else:
                    i += 1

            logger.info(f"Downloaded {len(self._tle_cache)} satellite TLEs")

        except requests.RequestException as e:
            logger.error(f"Failed to download TLE data: {e}")

            # Fall back to a minimal set of known satellites
            logger.warning("Using fallback TLE data for ISS")
            self._add_fallback_tle()

    def _add_fallback_tle(self) -> None:
        """Add fallback TLE data for ISS in case download fails"""
        # ISS TLE (updated periodically - this is a sample)
        iss_tle = TLEData(
            "ISS",
            "1 25544U 98067A   24010.50000000  .00010000  00000-0  18000-3 0  9998",
            "2 25544  51.6416 250.0000 0005000 200.0000 150.0000 15.50000000420000"
        )
        self._tle_cache["ISS"] = iss_tle

        # Add a few more fallback TLEs for common satellites
        # These are approximate TLEs for demonstration
        fallback_tles = [
            ("NOAA-18",
             "1 28654U 05018A   24010.50000000  .00000000  00000-0  00000-0 0  9998",
             "2 28654  99.1000 200.0000 0014000 100.0000 260.0000 14.10000000450000"),
            ("NOAA-19",
             "1 33591U 09005A   24010.50000000  .00000000  00000-0  00000-0 0  9998",
             "2 33591  99.1000 200.0000 0014000 100.0000 260.0000 14.10000000450000"),
        ]

        for name, line1, line2 in fallback_tles:
            self._tle_cache[name] = TLEData(name, line1, line2)
