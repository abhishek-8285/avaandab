import React from 'react';
import { StyleSheet, Text, View, TouchableOpacity } from 'react-native';
import { StatusBar } from 'expo-status-bar';
import { MaterialCommunityIcons } from '@expo/vector-icons';
import { LiveDriverTrackingMap } from './LiveDriverTrackingMap';
import { Colors, Font, Radius, Spacing } from '../constants/theme';

interface ActiveNavigationScreenProps {
  onArriveAtStop: () => void;
  onMenuToggle?: () => void;
}

export function ActiveNavigationScreen({ onArriveAtStop, onMenuToggle }: ActiveNavigationScreenProps) {
  return (
    <View style={styles.container}>
      <StatusBar style="light" />

      {/* Map background */}
      <View style={styles.mapContainer}>
        <LiveDriverTrackingMap
          driverLatitude={18.5255}
          driverLongitude={73.8520}
        />
      </View>

      {/* Top dark HUD bar */}
      <View style={styles.topAppBar}>
        <TouchableOpacity style={styles.iconBtn} onPress={onMenuToggle}>
          <MaterialCommunityIcons name="menu" size={18} color={Colors.textOnChrome} />
        </TouchableOpacity>

        <View style={styles.brandBlock}>
          <Text style={styles.brandTitle}>NAV</Text>
          <Text style={styles.brandSub}>TRP-8492 · LIVE</Text>
        </View>

        <TouchableOpacity style={styles.iconBtn}>
          <MaterialCommunityIcons name="bell-outline" size={16} color={Colors.textOnChrome} />
        </TouchableOpacity>
      </View>

      {/* Turn instruction HUD card */}
      <View style={styles.instructionContainer}>
        <View style={styles.turnCard}>
          <View style={styles.turnIconBox}>
            <MaterialCommunityIcons name="arrow-top-left" size={28} color={Colors.textOnPrimary} />
          </View>

          <View style={styles.turnTextContainer}>
            <Text style={styles.turnDistance}>0.5 MI</Text>
            <Text style={styles.turnTitle}>LEFT TURN</Text>
            <Text style={styles.turnSubtitle}>St Dunstan's Hill</Text>
          </View>
        </View>

        {/* Speed HUD */}
        <View style={styles.speedRow}>
          <View style={styles.speedBadge}>
            <Text style={styles.currentSpeed}>32</Text>
            <Text style={styles.speedUnit}>MPH</Text>
          </View>
          <View style={styles.speedLimitCircle}>
            <Text style={styles.speedLimitText}>30</Text>
          </View>
        </View>
      </View>

      {/* Bottom delivery card */}
      <View style={styles.bottomCardContainer}>
        <View style={styles.bottomCard}>
          <View style={styles.cardHeader}>
            <View style={styles.stopInfo}>
              <View style={styles.indicatorRow}>
                <View style={styles.greenDot} />
                <Text style={styles.stopLabel}>NEXT STOP · 02/04</Text>
              </View>
              <Text style={styles.stopAddress} numberOfLines={1}>
                4 St Dunstan's Hill
              </Text>
              <Text style={styles.stopRef}>REF #ORD-7492-X</Text>
            </View>

            <View style={styles.etaContainer}>
              <Text style={styles.etaLabel}>ETA</Text>
              <Text style={styles.etaTime}>12m</Text>
              <Text style={styles.etaDistance}>7.0 km</Text>
            </View>
          </View>

          <TouchableOpacity
            style={styles.arriveBtn}
            activeOpacity={0.88}
            onPress={onArriveAtStop}
          >
            <MaterialCommunityIcons name="map-marker-check" size={16} color={Colors.textOnPrimary} />
            <Text style={styles.arriveBtnText}>ARRIVE AT STOP</Text>
          </TouchableOpacity>
        </View>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: Colors.chrome,
  },
  mapContainer: {
    ...StyleSheet.absoluteFillObject,
  },
  topAppBar: {
    position: 'absolute',
    top: 48,
    left: Spacing.md,
    right: Spacing.md,
    height: 48,
    backgroundColor: Colors.chrome,
    borderRadius: Radius.md,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 10,
    borderWidth: 1,
    borderColor: Colors.chromeBorder,
    zIndex: 50,
  },
  iconBtn: {
    width: 32,
    height: 32,
    borderRadius: Radius.sm,
    backgroundColor: Colors.chromeLight,
    alignItems: 'center',
    justifyContent: 'center',
  },
  brandBlock: {
    alignItems: 'center',
  },
  brandTitle: {
    fontSize: 12,
    fontWeight: '900',
    color: Colors.textOnChrome,
    letterSpacing: 2,
    fontFamily: Font.mono,
  },
  brandSub: {
    fontSize: 8,
    fontWeight: '700',
    color: Colors.primary,
    letterSpacing: 1,
    fontFamily: Font.mono,
    marginTop: 2,
  },
  instructionContainer: {
    position: 'absolute',
    top: 108,
    left: Spacing.md,
    right: Spacing.md,
    zIndex: 40,
  },
  turnCard: {
    backgroundColor: Colors.chrome,
    borderRadius: Radius.md,
    padding: Spacing.md,
    flexDirection: 'row',
    alignItems: 'center',
    borderWidth: 1,
    borderColor: Colors.chromeBorder,
  },
  turnIconBox: {
    width: 44,
    height: 44,
    borderRadius: Radius.sm,
    backgroundColor: Colors.primary,
    alignItems: 'center',
    justifyContent: 'center',
    marginRight: Spacing.md,
  },
  turnTextContainer: {
    flex: 1,
  },
  turnDistance: {
    fontSize: 10,
    fontWeight: '700',
    color: Colors.primary,
    letterSpacing: 1,
    fontFamily: Font.mono,
  },
  turnTitle: {
    fontSize: 16,
    fontWeight: '900',
    color: Colors.textOnChrome,
    letterSpacing: 1,
    fontFamily: Font.mono,
    marginTop: 2,
  },
  turnSubtitle: {
    fontSize: 11,
    color: Colors.textOnChromeMuted,
    marginTop: 2,
  },
  speedRow: {
    flexDirection: 'row',
    justifyContent: 'flex-end',
    alignItems: 'center',
    marginTop: 8,
    gap: 8,
  },
  speedBadge: {
    backgroundColor: Colors.chrome,
    borderRadius: Radius.sm,
    paddingHorizontal: 10,
    paddingVertical: 6,
    flexDirection: 'row',
    alignItems: 'baseline',
    gap: 4,
    borderWidth: 1,
    borderColor: Colors.chromeBorder,
  },
  currentSpeed: {
    fontSize: 16,
    fontWeight: '900',
    color: Colors.textOnChrome,
    fontFamily: Font.mono,
  },
  speedUnit: {
    fontSize: 9,
    color: Colors.textOnChromeMuted,
    fontWeight: '700',
    letterSpacing: 1,
    fontFamily: Font.mono,
  },
  speedLimitCircle: {
    width: 32,
    height: 32,
    borderRadius: 16,
    borderWidth: 2,
    borderColor: Colors.danger,
    backgroundColor: Colors.surface,
    alignItems: 'center',
    justifyContent: 'center',
  },
  speedLimitText: {
    fontSize: 12,
    fontWeight: '900',
    color: Colors.danger,
    fontFamily: Font.mono,
  },
  bottomCardContainer: {
    position: 'absolute',
    bottom: Spacing.lg,
    left: Spacing.md,
    right: Spacing.md,
    zIndex: 40,
  },
  bottomCard: {
    backgroundColor: Colors.surface,
    borderRadius: Radius.md,
    padding: Spacing.lg,
    borderWidth: 1,
    borderColor: Colors.border,
  },
  cardHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'flex-start',
    marginBottom: Spacing.md,
  },
  stopInfo: {
    flex: 1,
    marginRight: Spacing.md,
  },
  indicatorRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
    marginBottom: 4,
  },
  greenDot: {
    width: 6,
    height: 6,
    borderRadius: 3,
    backgroundColor: Colors.primary,
  },
  stopLabel: {
    fontSize: 9,
    fontWeight: '700',
    color: Colors.textSecondary,
    letterSpacing: 1,
    fontFamily: Font.mono,
  },
  stopAddress: {
    fontSize: 15,
    fontWeight: '800',
    color: Colors.textPrimary,
    marginTop: 2,
  },
  stopRef: {
    fontSize: 10,
    color: Colors.textMuted,
    fontFamily: Font.mono,
    marginTop: 2,
    letterSpacing: 0.5,
  },
  etaContainer: {
    alignItems: 'flex-end',
    borderLeftWidth: 1,
    borderColor: Colors.border,
    paddingLeft: Spacing.md,
  },
  etaLabel: {
    fontSize: 9,
    fontWeight: '700',
    color: Colors.textMuted,
    letterSpacing: 1,
    fontFamily: Font.mono,
  },
  etaTime: {
    fontSize: 20,
    fontWeight: '900',
    color: Colors.primary,
    fontFamily: Font.mono,
    marginTop: 2,
  },
  etaDistance: {
    fontSize: 10,
    color: Colors.textSecondary,
    fontFamily: Font.mono,
    marginTop: 2,
  },
  arriveBtn: {
    height: 48,
    backgroundColor: Colors.primary,
    borderRadius: Radius.md,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 8,
  },
  arriveBtnText: {
    color: Colors.textOnPrimary,
    fontSize: 12,
    fontWeight: '800',
    letterSpacing: 1.5,
    fontFamily: Font.mono,
  },
});
