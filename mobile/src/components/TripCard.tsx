import React from 'react';
import { StyleSheet, Text, View, ActivityIndicator } from 'react-native';
import { Colors } from '../constants/theme';

interface TripCardProps {
  tripNumber: string;
  driverName: string;
  vehiclePlate: string;
  origin: string;
  destination: string;
  status: 'PENDING' | 'IN_TRANSIT' | 'COMPLETED' | 'CANCELLED';
  startTime: string;
  onPress?: () => void;
}

export const TripCard: React.FC<TripCardProps> = ({
  tripNumber,
  driverName,
  vehiclePlate,
  origin,
  destination,
  status,
  startTime,
}) => {
  const getStatusBadgeStyle = () => {
    switch (status) {
      case 'IN_TRANSIT':
        return { bg: Colors.successBg, text: Colors.success, label: 'IN TRANSIT' };
      case 'PENDING':
        return { bg: Colors.warningBg, text: Colors.warning, label: 'PENDING' };
      case 'COMPLETED':
        return { bg: Colors.primaryLight, text: Colors.primary, label: 'COMPLETED' };
      default:
        return { bg: Colors.dangerBg, text: Colors.danger, label: status };
    }
  };

  const badge = getStatusBadgeStyle();

  return (
    <View style={styles.card}>
      <View style={styles.header}>
        <Text style={styles.tripNumber}>{tripNumber}</Text>
        <View style={[styles.badge, { backgroundColor: badge.bg }]}>
          <Text style={[styles.badgeText, { color: badge.text }]}>{badge.label}</Text>
        </View>
      </View>

      <Text style={styles.label}>Driver & Unit</Text>
      <Text style={styles.value}>{driverName} • {vehiclePlate}</Text>

      <View style={styles.divider} />

      <View style={styles.routeContainer}>
        <Text style={styles.locationText}>📍 {origin}</Text>
        <Text style={styles.arrowText}>↓ Departure: {startTime}</Text>
        <Text style={styles.locationText}>🏁 {destination}</Text>
      </View>
    </View>
  );
};

export const SkeletonLoader = () => (
  <View style={[styles.card, { opacity: 0.6 }]}>
    <ActivityIndicator size="small" color={Colors.primary} />
  </View>
);

const styles = StyleSheet.create({
  card: {
    backgroundColor: Colors.surface,
    borderRadius: 14,
    padding: 16,
    borderWidth: 1,
    borderColor: Colors.border,
    marginBottom: 14,
    shadowColor: Colors.textPrimary,
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.04,
    shadowRadius: 6,
    elevation: 2,
  },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 10,
  },
  tripNumber: {
    fontSize: 17,
    fontWeight: '800',
    color: Colors.textPrimary,
  },
  badge: {
    paddingHorizontal: 10,
    paddingVertical: 4,
    borderRadius: 6,
  },
  badgeText: {
    fontSize: 11,
    fontWeight: '700',
    letterSpacing: 0.5,
  },
  label: {
    fontSize: 12,
    fontWeight: '500',
    color: Colors.textMuted,
    marginTop: 2,
  },
  value: {
    fontSize: 14,
    fontWeight: '600',
    color: Colors.textSecondary,
    marginTop: 2,
  },
  divider: {
    height: 1,
    backgroundColor: Colors.borderLight,
    marginVertical: 12,
  },
  routeContainer: {
    gap: 6,
  },
  locationText: {
    fontSize: 13,
    fontWeight: '600',
    color: Colors.textPrimary,
  },
  arrowText: {
    fontSize: 11,
    fontWeight: '500',
    color: Colors.textMuted,
    marginLeft: 18,
  },
});
