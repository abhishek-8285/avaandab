import React from 'react';
import { StyleSheet, View, Text } from 'react-native';
import MapView, { Marker, Polyline, PROVIDER_DEFAULT } from 'react-native-maps';
import { Colors } from '../constants/theme';

interface LiveDriverTrackingMapProps {
  driverLatitude: number;
  driverLongitude: number;
  pickupLatitude?: number;
  pickupLongitude?: number;
  destinationLatitude?: number;
  destinationLongitude?: number;
}

export function LiveDriverTrackingMap({
  driverLatitude,
  driverLongitude,
  pickupLatitude = 18.5204,
  pickupLongitude = 73.8567,
  destinationLatitude = 18.5308,
  destinationLongitude = 73.8474,
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
        {/* Active Route Polyline */}
        <Polyline
          coordinates={routeCoordinates}
          strokeColor="#0284c7"
          strokeWidth={4}
          lineDashPattern={[1]}
        />

        {/* Pickup Marker */}
        <Marker
          coordinate={{ latitude: pickupLatitude, longitude: pickupLongitude }}
          title="Pickup Location"
          description="Mumbai Port Terminal 2"
          pinColor="#10b981"
        />

        {/* Live Driver Marker (Uber Vehicle Position) */}
        <Marker
          coordinate={{ latitude: driverLatitude, longitude: driverLongitude }}
          title="Driver (Vehicle #TRK-9942)"
          description="Live Fleet Position"
          pinColor="#00685f"
        />

        {/* Destination Marker */}
        <Marker
          coordinate={{ latitude: destinationLatitude, longitude: destinationLongitude }}
          title="Destination Warehouse"
          description="Pune Logistics Hub B"
          pinColor="#ef4444"
        />
      </MapView>

      <View style={styles.statusBanner}>
        <View style={styles.statusDot} />
        <Text style={styles.statusText}>Uber-Style Live Dispatch Tracking Active</Text>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    height: 240,
    width: '100%',
    borderRadius: 12,
    overflow: 'hidden',
    marginTop: 10,
    borderWidth: 1,
    borderColor: Colors.border,
  },
  map: {
    ...StyleSheet.absoluteFillObject,
  },
  statusBanner: {
    position: 'absolute',
    top: 10,
    left: 10,
    right: 10,
    backgroundColor: 'rgba(15, 23, 42, 0.88)',
    borderRadius: 8,
    paddingVertical: 6,
    paddingHorizontal: 10,
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
  },
  statusDot: {
    width: 8,
    height: 8,
    borderRadius: 4,
    backgroundColor: '#10b981',
  },
  statusText: {
    color: '#ffffff',
    fontSize: 11,
    fontWeight: '700',
  },
});
