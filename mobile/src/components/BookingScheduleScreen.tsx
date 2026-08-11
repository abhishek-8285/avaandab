import React from 'react';
import { StyleSheet, Text, View, Image, TouchableOpacity } from 'react-native';
import { StatusBar } from 'expo-status-bar';
import { Colors } from '../constants/theme';

interface BookingScheduleScreenProps {
  onNext: () => void;
  onBack: () => void;
}

export function BookingScheduleScreen({ onNext, onBack }: BookingScheduleScreenProps) {
  return (
    <View style={styles.container}>
      <StatusBar style="dark" />
      
      {/* Header */}
      <View style={styles.header}>
        <TouchableOpacity onPress={onBack} style={styles.iconButton}>
          <Text style={styles.backArrow}>←</Text>
        </TouchableOpacity>
        <Text style={styles.headerTitle}>Booking Schedule</Text>
        <View style={{ width: 36 }} />
      </View>

      {/* Center Phone Mockup Section */}
      <View style={styles.heroContainer}>
        <View style={styles.phoneFrame}>
          <Image
            source={require('../../assets/booking_schedule.png')}
            style={styles.mockupImage}
            resizeMode="cover"
          />
        </View>
      </View>

      {/* Bottom Sheet Card */}
      <View style={styles.bottomCard}>
        <Text style={styles.headline}>
          Effortlessly Monitor Your{'\n'}
          <Text style={styles.highlightText}>Booking Schedule</Text>
        </Text>

        <Text style={styles.description}>
          Stay on top of your daily shifts with our intuitive calendar view and real-time ride tracking features.
        </Text>

        {/* Carousel Controls */}
        <View style={styles.footerRow}>
          <TouchableOpacity style={styles.iconButton} onPress={onBack}>
            <Text style={styles.controlArrow}>←</Text>
          </TouchableOpacity>

          <View style={styles.indicators}>
            <View style={styles.dot} />
            <View style={[styles.dot, styles.dotActive]} />
            <View style={styles.dot} />
          </View>

          <TouchableOpacity style={styles.nextButton} activeOpacity={0.85} onPress={onNext}>
            <Text style={styles.nextButtonText}>→</Text>
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
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingHorizontal: 24,
    paddingTop: 54,
    paddingBottom: 8,
    zIndex: 20,
  },
  headerTitle: {
    fontSize: 16,
    fontWeight: '600',
    color: '#0f172a',
  },
  iconButton: {
    width: 36,
    height: 36,
    borderRadius: 18,
    borderWidth: 1,
    borderColor: '#cbd5e1',
    alignItems: 'center',
    justifyContent: 'center',
  },
  backArrow: {
    fontSize: 18,
    color: '#0f172a',
    marginTop: -2,
  },
  controlArrow: {
    fontSize: 18,
    color: '#64748b',
    marginTop: -2,
  },
  heroContainer: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: 32,
  },
  phoneFrame: {
    width: 240,
    height: 420,
    borderRadius: 36,
    overflow: 'hidden',
    borderWidth: 6,
    borderColor: '#0f172a',
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 12 },
    shadowOpacity: 0.15,
    shadowRadius: 16,
    elevation: 10,
    backgroundColor: '#ffffff',
  },
  mockupImage: {
    width: '100%',
    height: '100%',
  },
  bottomCard: {
    backgroundColor: '#ffffff',
    borderTopLeftRadius: 40,
    borderTopRightRadius: 40,
    paddingHorizontal: 32,
    paddingTop: 32,
    paddingBottom: 36,
    alignItems: 'center',
    shadowColor: '#000',
    shadowOffset: { width: 0, height: -6 },
    shadowOpacity: 0.05,
    shadowRadius: 16,
    elevation: 12,
  },
  headline: {
    fontSize: 22,
    fontWeight: '700',
    color: '#0f172a',
    textAlign: 'center',
    lineHeight: 30,
    marginBottom: 12,
  },
  highlightText: {
    color: Colors.primary,
  },
  description: {
    fontSize: 13,
    color: '#64748b',
    textAlign: 'center',
    lineHeight: 20,
    marginBottom: 28,
  },
  footerRow: {
    width: '100%',
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  indicators: {
    flexDirection: 'row',
    gap: 8,
  },
  dot: {
    width: 8,
    height: 8,
    borderRadius: 4,
    backgroundColor: '#e2e8f0',
  },
  dotActive: {
    width: 24,
    backgroundColor: Colors.primary,
  },
  nextButton: {
    width: 48,
    height: 48,
    borderRadius: 24,
    backgroundColor: Colors.primary,
    alignItems: 'center',
    justifyContent: 'center',
    shadowColor: Colors.primary,
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.3,
    shadowRadius: 6,
    elevation: 6,
  },
  nextButtonText: {
    color: '#ffffff',
    fontSize: 22,
    fontWeight: '600',
    marginTop: -2,
  },
});
