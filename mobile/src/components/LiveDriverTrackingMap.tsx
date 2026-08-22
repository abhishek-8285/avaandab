import React from 'react';
import { StyleSheet, View, Text } from 'react-native';
import MapView, { Marker, Polyline, PROVIDER_DEFAULT } from 'react-native-maps';
import { Colors, Font, Radius } from '../constants/theme';
import {
  DEFAULT_DESTINATION_LATITUDE,
  DEFAULT_DESTINATION_LONGITUDE,
  DEFAULT_LATITUDE,
  DEFAULT_LONGITUDE,
} from '../constants/network';

interface LiveDriverTrackingMapProps {
  driverLatitude: number;
  driverLongitude: number;
  pickupLatitude?: number;
  pickupLongitude?: number;
  destinationLatitude?: number;
  destinationLongitude?: number;
  /** Real trip labels; falls back to neutral wording when unknown. */
  pickupLabel?: string;
  destinationLabel?: string;
  vehicleLabel?: string;
}

export function LiveDriverTrackingMap({
  driverLatitude,
  driverLongitude,
  pickupLatitude = DEFAULT_LATITUDE,
  pickupLongitude = DEFAULT_LONGITUDE,
  destinationLatitude = DEFAULT_DESTINATION_LATITUDE,
  destinationLongitude = DEFAULT_DESTINATION_LONGITUDE,
  pickupLabel,
  destinationLabel,
  vehicleLabel,
}: LiveDriverTrackingMapProps) {
  const routeCoordinates = [
    { latitude: pickupLatitude, longitude: pickupLongitude },
    { latitude: driverLatitude, longitude: driverLongitude },
    { latitude: destinationLatitude, longitude: destinationLongitude },
  ];

  return (
    <View style={styles.container}>
      <MapView
        style={styles.map}
        provider={PROVIDER_DEFAULT}
        initialRegion={{
          latitude: driverLatitude,
          longitude: driverLongitude,
          latitudeDelta: 0.04,
          longitudeDelta: 0.04,
        }}
        region={{
          latitude: driverLatitude,
          longitude: driverLongitude,
          latitudeDelta: 0.04,
          longitudeDelta: 0.04,
        }}
      >
        <Polyline
          coordinates={routeCoordinates}
          strokeColor={Colors.primary}
          strokeWidth={4}
          lineDashPattern={[1]}
        />

        <Marker
          coordinate={{ latitude: pickupLatitude, longitude: pickupLongitude }}
          title="Pickup Location"
          description={pickupLabel || 'Pickup point (location unknown)'}
          pinColor={Colors.success}
        />

        <Marker
          coordinate={{ latitude: driverLatitude, longitude: driverLongitude }}
          title={vehicleLabel ? `Driver (${vehicleLabel})` : 'Driver'}
          description="Live Fleet Position"
          pinColor={Colors.primary}
        />

        <Marker
          coordinate={{ latitude: destinationLatitude, longitude: destinationLongitude }}
          title="Destination Warehouse"
          description={destinationLabel || 'Destination (location unknown)'}
          pinColor={Colors.danger}
        />
      </MapView>

      <View style={styles.statusBanner}>
        <View style={styles.statusDot} />
        <Text style={styles.statusText}>LIVE · GPS TELEMETRY</Text>
        <Text style={styles.coordsText}>
          {driverLatitude.toFixed(4)}°N · {driverLongitude.toFixed(4)}°E
        </Text>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    height: 240,
    width: '100%',
    borderRadius: Radius.md,
    overflow: 'hidden',
    marginTop: 8,
    borderWidth: 1,
    borderColor: Colors.border,
  },
  map: {
    ...StyleSheet.absoluteFillObject,
  },
  statusBanner: {
    position: 'absolute',
    top: 8,
    left: 8,
    right: 8,
    backgroundColor: Colors.chrome,
    borderRadius: Radius.sm,
    paddingVertical: 6,
    paddingHorizontal: 10,
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
  },
  statusDot: {
    width: 6,
    height: 6,
    borderRadius: 3,
    backgroundColor: '#22c55e',
  },
  statusText: {
    color: Colors.textOnChrome,
    fontSize: 10,
    fontWeight: '800',
    letterSpacing: 1,
    fontFamily: Font.mono,
  },
  coordsText: {
    color: Colors.textOnChromeMuted,
    fontSize: 10,
    fontWeight: '600',
    fontFamily: Font.mono,
    marginLeft: 'auto',
  },
});
