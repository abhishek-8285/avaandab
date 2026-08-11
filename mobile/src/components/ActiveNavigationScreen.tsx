import React from 'react';
import { StyleSheet, Text, View, TouchableOpacity, Alert } from 'react-native';
import { StatusBar } from 'expo-status-bar';
import { MaterialCommunityIcons } from '@expo/vector-icons';
import { LiveDriverTrackingMap } from './LiveDriverTrackingMap';
import { Colors } from '../constants/theme';

interface ActiveNavigationScreenProps {
  onArriveAtStop: () => void;
  onMenuToggle?: () => void;
}

export function ActiveNavigationScreen({ onArriveAtStop, onMenuToggle }: ActiveNavigationScreenProps) {
  return (
    <View style={styles.container}>
      <StatusBar style="dark" />

      {/* Map Background with Interactive Polyline Navigation */}
      <View style={styles.mapContainer}>
        <LiveDriverTrackingMap
          driverLatitude={18.5255}
          driverLongitude={73.8520}
        />
      </View>

      {/* Top Glassmorphism App Bar */}
      <View style={styles.topAppBar}>
        <TouchableOpacity style={styles.iconCircle} onPress={onMenuToggle}>
          <MaterialCommunityIcons name="menu" size={24} color={Colors.primary} />
        </TouchableOpacity>

        <Text style={styles.brandTitle}>Avandab Navigation</Text>

        <TouchableOpacity style={styles.iconCircle}>
          <MaterialCommunityIcons name="bell-outline" size={22} color="#0b1c30" />
        </TouchableOpacity>
      </View>

      {/* Floating Turn Instruction Card */}
      <View style={styles.instructionContainer}>
        <View style={styles.turnCard}>
          <View style={styles.turnIconBox}>
            <MaterialCommunityIcons name="arrow-top-left" size={32} color={Colors.primary} />
          </View>

          <View style={styles.turnTextContainer}>
            <Text style={styles.turnTitle}>Left turn in 0.5 mi</Text>
            <Text style={styles.turnSubtitle}>onto St Dunstan's Hill</Text>
          </View>
        </View>

        {/* Speed / Limit Badge */}
        <View style={styles.speedRow}>
          <View style={styles.speedBadge}>
            <Text style={styles.currentSpeed}>32</Text>
            <Text style={styles.speedUnit}>mph</Text>
            <View style={styles.speedLimitCircle}>
              <Text style={styles.speedLimitText}>30</Text>
            </View>
          </View>
        </View>
      </View>

      {/* Bottom Floating Delivery Card */}
      <View style={styles.bottomCardContainer}>
        <View style={styles.bottomCard}>
          <View style={styles.cardHeader}>
            <View style={styles.stopInfo}>
              <View style={styles.indicatorRow}>
                <View style={styles.greenDot} />
                <Text style={styles.stopLabel}>NEXT STOP</Text>
              </View>
              <Text style={styles.stopAddress} numberOfLines={1}>
                4 St Dunstan's Hill
              </Text>
            </View>

            <View style={styles.etaContainer}>
              <Text style={styles.etaTime}>
                12<Text style={styles.etaMin}>min</Text>
              </Text>
              <Text style={styles.etaDistance}>7 km</Text>
            </View>
          </View>

          <TouchableOpacity
            style={styles.arriveBtn}
            activeOpacity={0.88}
            onPress={onArriveAtStop}
          >
            <MaterialCommunityIcons name="map-marker-check" size={22} color="#ffffff" />
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
    backgroundColor: '#f8fafc',
  },
  mapContainer: {
    ...StyleSheet.absoluteFillObject,
  },
  topAppBar: {
    position: 'absolute',
    top: 50,
    left: 16,
    right: 16,
    height: 52,
    backgroundColor: 'rgba(255, 255, 255, 0.92)',
    borderRadius: 16,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 12,
    borderWidth: 1,
    borderColor: 'rgba(226, 232, 240, 0.8)',
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.08,
    shadowRadius: 12,
    elevation: 4,
    zIndex: 50,
  },
  iconCircle: {
    width: 38,
    height: 38,
    borderRadius: 19,
    backgroundColor: '#f4fffc',
    alignItems: 'center',
    justifyContent: 'center',
  },
  brandTitle: {
    fontSize: 16,
    fontWeight: '700',
    color: Colors.primary,
  },
  instructionContainer: {
    position: 'absolute',
    top: 114,
    left: 16,
    right: 16,
    zIndex: 40,
  },
  turnCard: {
    backgroundColor: 'rgba(255, 255, 255, 0.95)',
    borderRadius: 16,
    padding: 16,
    flexDirection: 'row',
    alignItems: 'center',
    borderWidth: 1,
    borderColor: 'rgba(226, 232, 240, 0.8)',
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.08,
    shadowRadius: 12,
    elevation: 4,
  },
  turnIconBox: {
    width: 48,
    height: 48,
    borderRadius: 12,
    backgroundColor: '#f4fffc',
    alignItems: 'center',
    justifyContent: 'center',
    marginRight: 14,
  },
  turnTextContainer: {
    flex: 1,
  },
  turnTitle: {
    fontSize: 20,
    fontWeight: '700',
    color: '#0b1c30',
  },
  turnSubtitle: {
    fontSize: 13,
    color: '#5c647a',
    marginTop: 2,
  },
  speedRow: {
    flexDirection: 'row',
    justifyContent: 'flex-end',
    marginTop: 10,
  },
  speedBadge: {
    backgroundColor: 'rgba(255, 255, 255, 0.95)',
    borderRadius: 12,
    paddingHorizontal: 12,
    paddingVertical: 6,
    flexDirection: 'row',
    alignItems: 'center',
    gap: 4,
    borderWidth: 1,
    borderColor: '#ffdad6',
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.06,
    shadowRadius: 6,
    elevation: 3,
  },
  currentSpeed: {
    fontSize: 18,
    fontWeight: '700',
    color: '#0b1c30',
  },
  speedUnit: {
    fontSize: 11,
    color: '#5c647a',
    marginRight: 4,
  },
  speedLimitCircle: {
    width: 24,
    height: 24,
    borderRadius: 12,
    borderWidth: 2,
    borderColor: '#ba1a1a',
    backgroundColor: '#ffffff',
    alignItems: 'center',
    justifyContent: 'center',
  },
  speedLimitText: {
    fontSize: 10,
    fontWeight: '700',
    color: '#0b1c30',
  },
  bottomCardContainer: {
    position: 'absolute',
    bottom: 24,
    left: 16,
    right: 16,
    zIndex: 40,
  },
  bottomCard: {
    backgroundColor: 'rgba(255, 255, 255, 0.96)',
    borderRadius: 20,
    padding: 20,
    borderWidth: 1,
    borderColor: 'rgba(226, 232, 240, 0.8)',
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 8 },
    shadowOpacity: 0.12,
    shadowRadius: 20,
    elevation: 6,
  },
  cardHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'flex-start',
    marginBottom: 18,
  },
  stopInfo: {
    flex: 1,
    marginRight: 12,
  },
  indicatorRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
    marginBottom: 4,
  },
  greenDot: {
    width: 8,
    height: 8,
    borderRadius: 4,
    backgroundColor: Colors.primary,
  },
  stopLabel: {
    fontSize: 11,
    fontWeight: '700',
    color: '#5c647a',
    letterSpacing: 0.8,
  },
  stopAddress: {
    fontSize: 18,
    fontWeight: '700',
    color: '#0b1c30',
  },
  etaContainer: {
    alignItems: 'flex-end',
  },
  etaTime: {
    fontSize: 22,
    fontWeight: '700',
    color: Colors.primary,
  },
  etaMin: {
    fontSize: 12,
    fontWeight: '600',
    color: Colors.primary,
  },
  etaDistance: {
    fontSize: 12,
    color: '#5c647a',
    marginTop: 2,
  },
  arriveBtn: {
    height: 52,
    backgroundColor: Colors.primary,
    borderRadius: 12,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 8,
    shadowColor: Colors.primary,
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.25,
    shadowRadius: 8,
    elevation: 4,
  },
  arriveBtnText: {
    color: '#ffffff',
    fontSize: 14,
    fontWeight: '700',
    letterSpacing: 0.5,
  },
});
