"""
Position Calculator for OrbitStream Satellite Simulator

Uses Skyfield library to calculate satellite positions from TLE data.
Provides realistic orbital mechanics for satellite position tracking.
"""

import logging
from dataclasses import dataclass
from datetime import datetime, timezone
from typing import Dict, Optional

from skyfield.api import EarthSatellite, load, wgs84
from skyfield.toposlib import GeographicPosition

from .tle_manager import TLEData, TLEManager

logger = logging.getLogger(__name__)


@dataclass
class PositionData:
    """
    Satellite position data at a specific time.

    Attributes:
        latitude: Latitude in degrees (-90 to 90)
        longitude: Longitude in degrees (-180 to 180)
        altitude_km: Altitude above Earth's surface in kilometers
        velocity_kmph: Orbital velocity in kilometers per hour
    """
    latitude: float
    longitude: float
    altitude_km: float
    velocity_kmph: float

    def __repr__(self) -> str:
        return (
            f"PositionData(lat={self.latitude:.4f}, lon={self.longitude:.4f}, "
            f"alt={self.altitude_km:.2f}km, vel={self.velocity_kmph:.2f}km/h)"
        )


class PositionCalculator:
    """
    Calculates satellite positions from TLE data using Skyfield.

    Uses SGP4/SDP4 orbital propagation models via the Skyfield library
    to compute accurate satellite positions and velocities.
    """

    # Earth's mean radius in kilometers (for reference)
    EARTH_RADIUS_KM = 6371.0

    def __init__(self, tle_data: Dict[str, TLEData]):
        """
        Initialize position calculator with TLE data.

        Args:
            tle_data: Dictionary mapping satellite names to TLEData objects
        """
        self.tle_data = tle_data
        self.satellites: Dict[str, EarthSatellite] = {}
        self.ts = load.timescale()

        # Load satellites from TLE data
        self._load_satellites()

    def _load_satellites(self) -> None:
        """Load TLE data into Skyfield EarthSatellite objects"""
        for name, tle in self.tle_data.items():
            try:
                satellite = EarthSatellite(tle.line1, tle.line2, name, self.ts)
                self.satellites[name] = satellite
            except Exception as e:
                logger.warning(f"Failed to load satellite {name}: {e}")

        logger.info(f"Loaded {len(self.satellites)} satellites for position calculation")

    def get_position(self, satellite_name: str, timestamp: datetime) -> Optional[PositionData]:
        """
        Calculate satellite position at a given time.

        Args:
            satellite_name: Name of the satellite (case-insensitive)
            timestamp: Time at which to calculate position

        Returns:
            PositionData object with position and velocity, or None if satellite not found
        """
        # Find satellite (case-insensitive)
        satellite = self._find_satellite(satellite_name)
        if satellite is None:
            logger.warning(f"Satellite {satellite_name} not found")
            return None

        try:
            # Convert timestamp to Skyfield time
            # Skyfield uses UTC internally, so we convert to UTC
            if timestamp.tzinfo is None:
                # Assume UTC if no timezone info
                timestamp = timestamp.replace(tzinfo=timezone.utc)
            else:
                # Convert to UTC
                timestamp = timestamp.astimezone(timezone.utc)

            # Create Skyfield time object
            t = self.ts.utc(
                timestamp.year,
                timestamp.month,
                timestamp.day,
                timestamp.hour,
                timestamp.minute,
                timestamp.second + timestamp.microsecond / 1_000_000
            )

            # Calculate position
            geocentric = satellite.at(t)

            # Get subpoint position (latitude, longitude, elevation)
            subpoint = wgs84.subpoint(geocentric)

            latitude = subpoint.latitude.degrees
            longitude = subpoint.longitude.degrees
            altitude_km = subpoint.elevation.km

            # Calculate velocity
            # Skyfield gives velocity as a 3D vector in km/s
            # We need to compute the magnitude (scalar speed)
            import numpy as np
            velocity_kms = geocentric.velocity.km_per_s
            velocity_kmph = float(np.linalg.norm(velocity_kms)) * 3600.0

            # Normalize longitude to -180 to 180 range
            if longitude > 180:
                longitude -= 360
            elif longitude < -180:
                longitude += 360

            return PositionData(
                latitude=latitude,
                longitude=longitude,
                altitude_km=altitude_km,
                velocity_kmph=velocity_kmph
            )

        except Exception as e:
            logger.error(f"Error calculating position for {satellite_name}: {e}")
            return None

    def get_velocity(self, satellite_name: str, timestamp: datetime) -> Optional[float]:
        """
        Calculate satellite velocity at a given time.

        Args:
            satellite_name: Name of the satellite (case-insensitive)
            timestamp: Time at which to calculate velocity

        Returns:
            Velocity in km/h, or None if satellite not found
        """
        position = self.get_position(satellite_name, timestamp)
        return position.velocity_kmph if position else None

    def is_satellite_visible(
        self,
        satellite_name: str,
        timestamp: datetime,
        observer_lat: float,
        observer_lon: float,
        observer_alt_km: float = 0.0,
        min_elevation_deg: float = 10.0
    ) -> Optional[bool]:
        """
        Check if a satellite is visible from an observer's location.

        Args:
            satellite_name: Name of the satellite
            timestamp: Time at which to check visibility
            observer_lat: Observer's latitude in degrees
            observer_lon: Observer's longitude in degrees
            observer_alt_km: Observer's altitude above sea level in km
            min_elevation_deg: Minimum elevation angle in degrees

        Returns:
            True if satellite is visible, False if not, None if error
        """
        satellite = self._find_satellite(satellite_name)
        if satellite is None:
            return None

        try:
            # Create observer position
            observer = wgs84.latlon(observer_lat, observer_lon, observer_alt_km)

            # Convert timestamp to Skyfield time
            if timestamp.tzinfo is None:
                timestamp = timestamp.replace(tzinfo=timezone.utc)
            else:
                timestamp = timestamp.astimezone(timezone.utc)

            t = self.ts.utc(
                timestamp.year,
                timestamp.month,
                timestamp.day,
                timestamp.hour,
                timestamp.minute,
                timestamp.second + timestamp.microsecond / 1_000_000
            )

            # Calculate satellite position relative to observer
            difference = satellite - observer
            topocentric = difference.at(t)

            # Get altitude, azimuth, and distance
            alt, az, distance = topocentric.altaz()

            # Check if above minimum elevation
            # Convert numpy bool to Python bool
            return bool(alt.degrees >= min_elevation_deg)

        except Exception as e:
            logger.error(f"Error checking visibility for {satellite_name}: {e}")
            return None

    def _find_satellite(self, satellite_name: str) -> Optional[EarthSatellite]:
        """Find satellite by name (case-insensitive)"""
        # Try exact match (case-insensitive)
        for name, satellite in self.satellites.items():
            if name.upper() == satellite_name.upper():
                return satellite

        # Try partial match
        for name, satellite in self.satellites.items():
            if satellite_name.upper() in name.upper():
                return satellite

        return None

    def get_available_satellites(self) -> list[str]:
        """Get list of available satellite names"""
        return list(self.satellites.keys())

    def get_orbital_period(self, satellite_name: str) -> Optional[float]:
        """
        Calculate the orbital period of a satellite.

        Args:
            satellite_name: Name of the satellite

        Returns:
            Orbital period in minutes, or None if not found
        """
        satellite = self._find_satellite(satellite_name)
        if satellite is None:
            return None

        try:
            # Extract mean motion from TLE line 2 (columns 53-63, 0-indexed: 52:63)
            # Mean motion is in revolutions per day
            tle = self._find_tle(satellite_name)
            if tle is None:
                return None

            # Extract mean motion from TLE line 2 (columns 53-63)
            mean_motion_rev_day = float(tle.line2[52:63])

            # Convert to period in minutes
            # Period = 1440 minutes / mean motion (revolutions per day)
            period_minutes = 1440.0 / mean_motion_rev_day

            return period_minutes

        except Exception as e:
            logger.error(f"Error calculating orbital period for {satellite_name}: {e}")
            return None

    def _find_tle(self, satellite_name: str) -> Optional[TLEData]:
        """Find TLE data for a satellite"""
        for name, tle in self.tle_data.items():
            if name.upper() == satellite_name.upper():
                return tle
            if satellite_name.upper() in name.upper():
                return tle
        return None


def create_position_manager() -> PositionCalculator:
    """
    Convenience function to create a position manager with TLE data.

    Returns:
        PositionCalculator instance with loaded TLE data
    """
    tle_manager = TLEManager()
    tle_data = tle_manager.load_tle_data()
    return PositionCalculator(tle_data)
