import React from 'react';
import { StyleSheet, Text, View, ActivityIndicator } from 'react-native';
import { MaterialCommunityIcons } from '@expo/vector-icons';
import { Colors, Font, Radius, Spacing } from '../constants/theme';

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
        return { bg: Colors.successBg, text: Colors.success, label: 'IN TRANSIT', dot: Colors.success };
      case 'PENDING':
        return { bg: Colors.warningBg, text: Colors.warning, label: 'PENDING', dot: Colors.warning };
      case 'COMPLETED':
        return { bg: Colors.primaryLight, text: Colors.primary, label: 'COMPLETED', dot: Colors.primary };
      default:
        return { bg: Colors.dangerBg, text: Colors.danger, label: status, dot: Colors.danger };
    }
  };

  const badge = getStatusBadgeStyle();

  return (
    <View style={styles.card}>
      <View style={styles.header}>
        <View style={styles.tripIdBlock}>
          <Text style={styles.tripLabel}>TRIP</Text>
          <Text style={styles.tripNumber}>{tripNumber}</Text>
        </View>
        <View style={[styles.badge, { backgroundColor: badge.bg }]}>
          <View style={[styles.badgeDot, { backgroundColor: badge.dot }]} />
          <Text style={[styles.badgeText, { color: badge.text }]}>{badge.label}</Text>
        </View>
      </View>

      <View style={styles.driverRow}>
        <MaterialCommunityIcons name="account-hard-hat" size={12} color={Colors.textMuted} />
        <Text style={styles.driverName}>{driverName}</Text>
        <View style={styles.plateChip}>
          <Text style={styles.plateText}>{vehiclePlate}</Text>
        </View>
      </View>

      <View style={styles.divider} />

      <View style={styles.routeContainer}>
        <View style={styles.routeRow}>
          <View style={[styles.routeDot, styles.routeDotOrigin]} />
          <Text style={styles.locationText} numberOfLines={1}>{origin}</Text>
        </View>
        <View style={styles.routeConnector} />
        <View style={styles.routeMeta}>
          <Text style={styles.departureText}>DEP {startTime}</Text>
        </View>
        <View style={styles.routeRow}>
          <View style={[styles.routeDot, styles.routeDotDest]} />
          <Text style={styles.locationText} numberOfLines={1}>{destination}</Text>
        </View>
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
    borderRadius: Radius.md,
    padding: Spacing.md,
    borderWidth: 1,
    borderColor: Colors.border,
    marginBottom: Spacing.md,
  },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'flex-start',
    marginBottom: Spacing.sm,
  },
  tripIdBlock: {
    flexDirection: 'row',
    alignItems: 'baseline',
    gap: 6,
  },
  tripLabel: {
    fontSize: 9,
    fontWeight: '700',
    color: Colors.textMuted,
    letterSpacing: 1,
    fontFamily: Font.mono,
  },
  tripNumber: {
    fontSize: 15,
    fontWeight: '900',
    color: Colors.textPrimary,
    fontFamily: Font.mono,
    letterSpacing: 0.5,
  },
  badge: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 4,
    paddingHorizontal: 8,
    paddingVertical: 3,
    borderRadius: Radius.sm,
  },
  badgeDot: {
    width: 5,
    height: 5,
    borderRadius: 2,
  },
  badgeText: {
    fontSize: 9,
    fontWeight: '800',
    letterSpacing: 0.5,
    fontFamily: Font.mono,
  },
  driverRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
  },
  driverName: {
    fontSize: 12,
    fontWeight: '600',
    color: Colors.textSecondary,
  },
  plateChip: {
    marginLeft: 'auto',
    backgroundColor: Colors.surfaceSecondary,
    borderWidth: 1,
    borderColor: Colors.border,
    borderRadius: Radius.sm,
    paddingHorizontal: 6,
    paddingVertical: 2,
  },
  plateText: {
    fontSize: 10,
    fontWeight: '800',
    color: Colors.textPrimary,
    fontFamily: Font.mono,
    letterSpacing: 0.5,
  },
  divider: {
    height: 1,
    backgroundColor: Colors.borderLight,
    marginVertical: Spacing.md,
  },
  routeContainer: {
    gap: 0,
  },
  routeRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 10,
  },
  routeDot: {
    width: 8,
    height: 8,
    borderRadius: 2,
  },
  routeDotOrigin: {
    backgroundColor: Colors.success,
  },
  routeDotDest: {
    backgroundColor: Colors.danger,
  },
  locationText: {
    fontSize: 12,
    fontWeight: '700',
    color: Colors.textPrimary,
    flex: 1,
  },
  routeConnector: {
    width: 1,
    height: 10,
    backgroundColor: Colors.border,
    marginLeft: 3.5,
  },
  routeMeta: {
    marginLeft: 18,
    marginVertical: 2,
  },
  departureText: {
    fontSize: 9,
    fontWeight: '700',
    color: Colors.textMuted,
    letterSpacing: 1,
    fontFamily: Font.mono,
  },
});
