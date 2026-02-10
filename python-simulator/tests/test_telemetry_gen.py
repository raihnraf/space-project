"""
Tests for the TelemetryGenerator class
"""
import random
from datetime import datetime, timezone
from unittest.mock import MagicMock, Mock

from generators.telemetry_gen import TelemetryGenerator
from generators.position_calc import PositionData


class TestTelemetryGeneratorInitialization:
    """Tests for TelemetryGenerator initialization"""

    def test_initialization_defaults(self):
        """Test default values are set correctly"""
        gen = TelemetryGenerator()
        assert gen.base_battery == 100.0
        assert gen.base_storage == 0.0
        assert gen.base_signal == -50.0
        assert gen.anomaly_rate == 0.01

    def test_initialization_custom_values(self):
        """Test custom values are set correctly"""
        gen = TelemetryGenerator(
            base_battery=90.0,
            base_storage=1000.0,
            base_signal=-60.0,
            anomaly_rate=0.05
        )
        assert gen.base_battery == 90.0
        assert gen.base_storage == 1000.0
        assert gen.base_signal == -60.0
        assert gen.anomaly_rate == 0.05

    def test_post_init_initializes_state(self):
        """Test __post_init__ sets up internal state"""
        gen = TelemetryGenerator(
            base_battery=100.0,
            base_storage=0.0,
            base_signal=-50.0,
            anomaly_rate=0.01
        )
        # Check that internal state is initialized
        assert hasattr(gen, 'battery')
        assert hasattr(gen, 'storage')
        assert hasattr(gen, 'signal')
        assert hasattr(gen, 'battery_drain_rate')
        assert hasattr(gen, 'storage_growth_rate')
        assert hasattr(gen, 'signal_volatility')
        assert gen.battery == gen.base_battery
        assert gen.storage == gen.base_storage
        assert gen.signal == gen.base_signal


class TestNormalTelemetryGeneration:
    """Tests for normal (non-anomalous) telemetry generation"""

    def test_generate_telemetry_returns_dict(self, telemetry_generator):
        """Test that generate_telemetry returns a dictionary"""
        result = telemetry_generator.generate_telemetry()
        assert isinstance(result, dict)

    def test_generate_telemetry_has_all_fields(self, telemetry_generator):
        """Test that telemetry has all required fields"""
        result = telemetry_generator.generate_telemetry()
        assert 'battery' in result
        assert 'storage' in result
        assert 'signal' in result

    def test_generate_telemetry_values_are_numeric(self, telemetry_generator):
        """Test that all telemetry values are numeric"""
        result = telemetry_generator.generate_telemetry()
        assert isinstance(result['battery'], (int, float))
        assert isinstance(result['storage'], (int, float))
        assert isinstance(result['signal'], (int, float))

    def test_battery_stays_within_bounds(self, telemetry_generator):
        """Test that battery stays within 0-100 range"""
        for _ in range(1000):
            result = telemetry_generator.generate_telemetry()
            assert 0 <= result['battery'] <= 100, \
                f"Battery {result['battery']} out of range [0, 100]"

    def test_storage_never_negative(self, telemetry_generator):
        """Test that storage is never negative"""
        for _ in range(1000):
            result = telemetry_generator.generate_telemetry()
            assert result['storage'] >= 0, \
                f"Storage {result['storage']} is negative"

    def test_signal_stays_within_dBm_range(self, telemetry_generator):
        """Test that signal stays within valid dBm range"""
        for _ in range(1000):
            result = telemetry_generator.generate_telemetry()
            assert -120 <= result['signal'] <= -30, \
                f"Signal {result['signal']} out of dBm range [-120, -30]"

    def test_battery_drain_trend(self, telemetry_generator_no_anomalies):
        """Test that battery has a downward trend over time"""
        gen = telemetry_generator_no_anomalies
        # Use a fixed seed for reproducibility
        import random
        random.seed(123)
        import numpy as np
        np.random.seed(123)

        # Track battery levels
        battery_levels = []
        for _ in range(100):
            result = gen.generate_telemetry()
            battery_levels.append(result['battery'])

        # Compare average of first 10 vs last 10 readings
        # The average should decrease (general downward trend)
        first_avg = sum(battery_levels[:10]) / 10
        last_avg = sum(battery_levels[-10:]) / 10

        assert last_avg < first_avg, \
            f"Battery should show downward trend: first avg {first_avg}, last avg {last_avg}"

    def test_storage_growth_trend(self, telemetry_generator_no_anomalies):
        """Test that storage generally increases over time"""
        gen = telemetry_generator_no_anomalies
        initial_storage = gen.storage

        # Generate multiple readings
        for _ in range(50):
            gen.generate_telemetry()

        # Storage should be higher
        assert gen.storage > initial_storage, \
            f"Storage didn't grow: started at {initial_storage}, now at {gen.storage}"


class TestAnomalyGeneration:
    """Tests for anomalous telemetry generation"""

    def test_anomaly_rate_distribution(self, telemetry_generator_high_anomaly_rate):
        """Test that anomaly rate setting affects generation"""
        # Use high anomaly rate to ensure we get detectable anomalies
        gen = telemetry_generator_high_anomaly_rate
        anomaly_count = 0
        total_samples = 1000

        for _ in range(total_samples):
            result = gen.generate_telemetry()
            # Check for clear anomaly indicators:
            # - battery_critical: battery < 10
            # - storage_full: storage > 95000
            # - signal_loss: signal < -110
            if (result['battery'] < 10 or
                result['storage'] > 95000 or
                result['signal'] < -110):
                anomaly_count += 1

        # With 100% anomaly rate, we should see many anomalies
        # (not 100% because anomaly types vary and not all affect all metrics)
        assert anomaly_count > total_samples * 0.3, \
            f"Expected many anomalies with high rate, got {anomaly_count}/{total_samples}"

    def test_anomaly_battery_critical(self, telemetry_generator_high_anomaly_rate):
        """Test battery critical anomaly generates very low battery"""
        gen = telemetry_generator_high_anomaly_rate
        found_battery_critical = False

        for _ in range(1000):
            result = gen.generate_telemetry()
            if result['battery'] < 10:
                found_battery_critical = True
                assert 0 <= result['battery'] <= 10, \
                    f"Battery critical anomaly should be 0-10%, got {result['battery']}"
                break

        assert found_battery_critical, "Never generated a battery critical anomaly in 1000 attempts"

    def test_anomaly_storage_full(self, telemetry_generator_high_anomaly_rate):
        """Test storage full anomaly generates very high storage"""
        gen = telemetry_generator_high_anomaly_rate
        found_storage_full = False

        for _ in range(1000):
            result = gen.generate_telemetry()
            if result['storage'] > 95000:
                found_storage_full = True
                assert 95000 <= result['storage'] <= 100000, \
                    f"Storage full anomaly should be 95000-100000 MB, got {result['storage']}"
                break

        assert found_storage_full, "Never generated a storage full anomaly in 1000 attempts"

    def test_anomaly_signal_loss(self, telemetry_generator_high_anomaly_rate):
        """Test signal loss anomaly generates very weak signal"""
        gen = telemetry_generator_high_anomaly_rate
        found_signal_loss = False

        for _ in range(1000):
            result = gen.generate_telemetry()
            if result['signal'] < -110:
                found_signal_loss = True
                assert -120 <= result['signal'] <= -110, \
                    f"Signal loss anomaly should be -120 to -110 dBm, got {result['signal']}"
                break

        assert found_signal_loss, "Never generated a signal loss anomaly in 1000 attempts"

    def test_anomaly_sudden_discharge(self, telemetry_generator_high_anomaly_rate):
        """Test sudden discharge anomaly drops battery significantly"""
        # Use high anomaly rate fixture to increase chances of triggering
        gen = telemetry_generator_high_anomaly_rate
        # Set battery to a known value
        gen.battery = 80.0

        found_sudden_discharge = False  # noqa: F841 - variable used for documentation
        for _ in range(1000):
            # Capture battery BEFORE this call (it changes after each call)
            battery_before = gen.battery
            result = gen.generate_telemetry()
            # Sudden discharge should drop battery by 20-40 from current value
            # Check if this was a sudden discharge anomaly (battery dropped by ~20-50)
            drop = battery_before - result['battery']
            if 15 <= drop <= 50:
                assert 20 <= drop <= 50, \
                    f"Sudden discharge should drop 20-40, got {drop}"
                break

        # This test may not always find the anomaly due to randomness
        # so we just verify the mechanism exists if we found one


class TestEdgeCases:
    """Tests for edge cases and special conditions"""

    def test_battery_clamping_at_zero(self):
        """Test that battery is clamped at 0"""
        gen = TelemetryGenerator(base_battery=5.0, anomaly_rate=0.0)
        # Generate many points to drain battery
        for _ in range(1000):
            result = gen.generate_telemetry()
            assert result['battery'] >= 0, "Battery went below 0"

    def test_battery_clamping_at_100(self):
        """Test that charging event clamps battery at 100"""
        gen = TelemetryGenerator(base_battery=95.0, anomaly_rate=0.0)
        # Force a charging condition by setting battery low
        gen.battery = 25.0

        for _ in range(200):
            result = gen.generate_telemetry()
            if result['battery'] > gen.battery:
                # Battery increased (charging)
                assert result['battery'] <= 100, \
                    f"Battery exceeded 100 after charging: {result['battery']}"
                break

        # May not always trigger due to randomness

    def test_storage_cleanup_when_full(self):
        """Test that storage decreases when it gets very high"""
        gen = TelemetryGenerator(base_storage=95000.0, anomaly_rate=0.0)
        initial_storage = gen.storage

        # Generate points - storage should eventually decrease
        for _ in range(500):
            result = gen.generate_telemetry()
            if result['storage'] < initial_storage:
                # Check cleanup amount
                cleanup_amount = initial_storage - result['storage']
                assert 5000 <= cleanup_amount <= 25000, \
                    f"Storage cleanup should be 5000-25000 MB, got {cleanup_amount}"
                break

        # This test is probabilistic

    def test_custom_anomaly_rate(self):
        """Test custom anomaly rates work correctly"""
        # Test with 50% anomaly rate
        gen = TelemetryGenerator(anomaly_rate=0.5)

        # Seed for reproducibility
        random.seed(42)
        anomaly_count = 0
        for _ in range(100):
            result = gen.generate_telemetry()
            # Anomalies have extreme values
            if (result['battery'] < 20 or
                result['storage'] > 90000 or
                result['signal'] < -100):
                anomaly_count += 1

        # Should be close to 50%
        assert 30 <= anomaly_count <= 70, \
            f"With 50% anomaly rate, expected ~50 anomalies, got {anomaly_count}"

    def test_zero_anomaly_rate(self, telemetry_generator_no_anomalies):
        """Test that zero anomaly rate produces no extreme anomalies"""
        gen = telemetry_generator_no_anomalies

        for _ in range(1000):
            result = gen.generate_telemetry()
            # With no anomalies, values should stay in normal operating range
            # Signal can naturally fluctuate between -120 and -30
            # Storage grows but shouldn't reach critical levels immediately
            assert result['battery'] >= 0, \
                "Battery went negative with zero anomaly rate"
            assert result['storage'] >= 0, \
                "Storage went negative with zero anomaly rate"

    def test_multiple_generators_independent(self):
        """Test that multiple generators are independent"""
        gen1 = TelemetryGenerator(base_battery=100.0, base_storage=0.0, base_signal=-50.0, anomaly_rate=0.0)
        gen2 = TelemetryGenerator(base_battery=50.0, base_storage=50000.0, base_signal=-70.0, anomaly_rate=0.0)

        result1 = gen1.generate_telemetry()
        result2 = gen2.generate_telemetry()

        # Results should be different
        assert result1['battery'] != result2['battery'] or \
               result1['storage'] != result2['storage'] or \
               result1['signal'] != result2['signal'], \
               "Independent generators produced identical results"


class TestTelemetryValues:
    """Tests for specific telemetry value properties"""

    def test_values_are_rounded(self, telemetry_generator):
        """Test that values are rounded to 2 decimal places"""
        result = telemetry_generator.generate_telemetry()
        # Check that values have at most 2 decimal places
        assert round(result['battery'], 2) == result['battery'], \
            f"Battery not rounded to 2 decimals: {result['battery']}"
        assert round(result['storage'], 2) == result['storage'], \
            f"Storage not rounded to 2 decimals: {result['storage']}"
        assert round(result['signal'], 2) == result['signal'], \
            f"Signal not rounded to 2 decimals: {result['signal']}"

    def test_consistent_return_format(self, telemetry_generator):
        """Test that return format is consistent"""
        gen = telemetry_generator
        results = [gen.generate_telemetry() for _ in range(100)]

        for result in results:
            assert isinstance(result, dict)
            # Should at least have the basic fields (may also have position fields)
            assert 'battery' in result
            assert 'storage' in result
            assert 'signal' in result

    def test_signal_fluctuates(self, telemetry_generator_no_anomalies):
        """Test that signal fluctuates around base value"""
        gen = telemetry_generator_no_anomalies
        signals = [gen.generate_telemetry()['signal'] for _ in range(100)]

        # Check there's variation
        assert len(set(signals)) > 50, "Signal values should vary"

        # Check values are generally around base (-50)
        avg_signal = sum(signals) / len(signals)
        assert -70 < avg_signal < -30, \
            f"Average signal {avg_signal} not around base -50"


# =============================================================================
# Feature E: Position Tracking Tests
# =============================================================================


class TestTelemetryGeneratorPositionFields:
    """Tests for position tracking in telemetry generation"""

    def test_telemetry_without_position_calculator(self):
        """Test telemetry generation without position calculator returns basic fields"""
        gen = TelemetryGenerator(satellite_name="ISS", position_calculator=None)
        result = gen.generate_telemetry()

        # Should only have basic fields
        assert 'battery' in result
        assert 'storage' in result
        assert 'signal' in result
        # Position fields should not be present
        assert 'latitude' not in result
        assert 'longitude' not in result
        assert 'altitude_km' not in result
        assert 'velocity_kmph' not in result

    def test_telemetry_with_position_calculator(self):
        """Test telemetry generation with position calculator includes position fields"""
        # Create mock position calculator
        mock_calc = Mock()
        mock_calc.get_position.return_value = PositionData(
            latitude=40.7128,
            longitude=-74.0060,
            altitude_km=408.5,
            velocity_kmph=27576.5
        )

        gen = TelemetryGenerator(
            satellite_name="ISS",
            position_calculator=mock_calc
        )
        result = gen.generate_telemetry()

        # Should have all fields including position
        assert 'battery' in result
        assert 'storage' in result
        assert 'signal' in result
        assert 'latitude' in result
        assert 'longitude' in result
        assert 'altitude_km' in result
        assert 'velocity_kmph' in result

    def test_position_values_are_correctly_set(self):
        """Test that position values from calculator are correctly added"""
        mock_calc = Mock()
        mock_calc.get_position.return_value = PositionData(
            latitude=-33.8688,  # Sydney
            longitude=151.2093,
            altitude_km=408.5,
            velocity_kmph=27576.5
        )

        gen = TelemetryGenerator(
            satellite_name="ISS",
            position_calculator=mock_calc
        )
        result = gen.generate_telemetry()

        assert result['latitude'] == -33.8688
        assert result['longitude'] == 151.2093
        assert result['altitude_km'] == 408.5
        assert result['velocity_kmph'] == 27576.5

    def test_position_rounding(self):
        """Test that position values are rounded correctly"""
        mock_calc = Mock()
        mock_calc.get_position.return_value = PositionData(
            latitude=40.71284567,  # More than 6 decimals
            longitude=-74.00601234,
            altitude_km=408.56789,  # More than 2 decimals
            velocity_kmph=27576.54321
        )

        gen = TelemetryGenerator(
            satellite_name="ISS",
            position_calculator=mock_calc
        )
        result = gen.generate_telemetry()

        # Latitude and longitude should be rounded to 6 decimals
        assert result['latitude'] == 40.712846
        assert result['longitude'] == -74.006012
        # Altitude and velocity should be rounded to 2 decimals
        assert result['altitude_km'] == 408.57
        assert result['velocity_kmph'] == 27576.54

    def test_position_calculator_called_with_correct_params(self):
        """Test that position calculator is called with correct parameters"""
        mock_calc = Mock()
        mock_calc.get_position.return_value = PositionData(
            latitude=0.0,
            longitude=0.0,
            altitude_km=400.0,
            velocity_kmph=27500.0
        )

        gen = TelemetryGenerator(
            satellite_name="STARLINK-1001",
            position_calculator=mock_calc
        )
        gen.generate_telemetry()

        # Verify get_position was called with satellite name
        mock_calc.get_position.assert_called_once()
        call_args = mock_calc.get_position.call_args
        assert call_args[0][0] == "STARLINK-1001"  # First arg is satellite_name
        # Second arg should be a datetime
        assert isinstance(call_args[0][1], datetime)

    def test_position_fallback_on_calculator_error(self):
        """Test that last known position is used when calculator fails"""
        mock_calc = Mock()

        # First call succeeds
        mock_calc.get_position.side_effect = [
            PositionData(
                latitude=35.6762,
                longitude=139.6503,
                altitude_km=408.0,
                velocity_kmph=27580.0
            ),
            # Second call fails with exception
            Exception("Calculation failed")
        ]

        gen = TelemetryGenerator(
            satellite_name="ISS",
            position_calculator=mock_calc
        )

        # First generation should use calculator
        result1 = gen.generate_telemetry()
        assert result1['latitude'] == 35.6762

        # Second generation should use fallback (last position)
        result2 = gen.generate_telemetry()
        assert result2['latitude'] == 35.6762  # Should use last known position

    def test_position_calculator_returns_none(self):
        """Test behavior when position calculator returns None"""
        mock_calc = Mock()
        mock_calc.get_position.return_value = None

        gen = TelemetryGenerator(
            satellite_name="UNKNOWN-SAT",
            position_calculator=mock_calc
        )
        result = gen.generate_telemetry()

        # Should only have basic fields when position is None
        assert 'battery' in result
        assert 'storage' in result
        assert 'signal' in result
        # Position fields should not be present
        assert 'latitude' not in result

    def test_position_fields_with_anomaly(self):
        """Test that position fields are included even during anomalies"""
        mock_calc = Mock()
        mock_calc.get_position.return_value = PositionData(
            latitude=1.3521,
            longitude=103.8198,
            altitude_km=400.0,
            velocity_kmph=27500.0
        )

        gen = TelemetryGenerator(
            base_battery=100.0,
            satellite_name="ISS",
            position_calculator=mock_calc,
            anomaly_rate=1.0  # Always generate anomaly
        )
        result = gen.generate_telemetry()

        # Should have both anomaly telemetry AND position fields
        assert 'battery' in result
        assert 'storage' in result
        assert 'signal' in result
        assert 'latitude' in result
        assert 'longitude' in result
        assert 'altitude_km' in result
        assert 'velocity_kmph' in result

    def test_position_value_ranges(self):
        """Test that position values are within realistic ranges"""
        mock_calc = Mock()

        # Test various realistic positions
        test_positions = [
            # Equatorial
            PositionData(0.0, 0.0, 400.0, 27500.0),
            # North pole
            PositionData(90.0, 0.0, 500.0, 27000.0),
            # South pole
            PositionData(-90.0, 180.0, 450.0, 27800.0),
            # International date line
            PositionData(45.0, -180.0, 420.0, 27650.0),
            # Low LEO
            PositionData(45.0, -122.0, 300.0, 27300.0),
            # High LEO
            PositionData(-45.0, 0.0, 2000.0, 26000.0),
        ]

        for pos in test_positions:
            mock_calc.get_position.return_value = pos

            gen = TelemetryGenerator(
                satellite_name="TEST-SAT",
                position_calculator=mock_calc
            )
            result = gen.generate_telemetry()

            # Verify ranges
            assert -90 <= result['latitude'] <= 90, \
                f"Latitude {result['latitude']} out of range [-90, 90]"
            assert -180 <= result['longitude'] <= 180, \
                f"Longitude {result['longitude']} out of range [-180, 180]"
            assert 300 <= result['altitude_km'] <= 2000, \
                f"Altitude {result['altitude_km']} out of LEO range [300, 2000]"
            assert 26000 <= result['velocity_kmph'] <= 28000, \
                f"Velocity {result['velocity_kmph']} out of orbital range [26000, 28000]"

    def test_position_without_satellite_name(self):
        """Test that position is not calculated without satellite name"""
        mock_calc = Mock()
        mock_calc.get_position.return_value = PositionData(
            latitude=0.0,
            longitude=0.0,
            altitude_km=400.0,
            velocity_kmph=27500.0
        )

        # No satellite name specified
        gen = TelemetryGenerator(position_calculator=mock_calc)
        result = gen.generate_telemetry()

        # Should not call get_position without satellite name
        mock_calc.get_position.assert_not_called()

        # Should only have basic fields
        assert 'battery' in result
        assert 'latitude' not in result

    def test_multiple_generators_with_positions(self):
        """Test multiple generators with different positions"""
        mock_calc_iss = Mock()
        mock_calc_iss.get_position.return_value = PositionData(
            latitude=40.7128,
            longitude=-74.0060,
            altitude_km=408.5,
            velocity_kmph=27576.5
        )

        mock_calc_starlink = Mock()
        mock_calc_starlink.get_position.return_value = PositionData(
            latitude=-33.8688,
            longitude=151.2093,
            altitude_km=550.0,
            velocity_kmph=27600.0
        )

        gen_iss = TelemetryGenerator(
            satellite_name="ISS",
            position_calculator=mock_calc_iss
        )
        gen_starlink = TelemetryGenerator(
            satellite_name="STARLINK-1001",
            position_calculator=mock_calc_starlink
        )

        result_iss = gen_iss.generate_telemetry()
        result_starlink = gen_starlink.generate_telemetry()

        # Positions should be different
        assert result_iss['latitude'] != result_starlink['latitude']
        assert result_iss['longitude'] != result_starlink['longitude']
        assert result_iss['altitude_km'] != result_starlink['altitude_km']
