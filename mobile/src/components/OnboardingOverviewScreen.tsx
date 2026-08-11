import React from 'react';
import { StyleSheet, Text, View, Image, TouchableOpacity } from 'react-native';
import { StatusBar } from 'expo-status-bar';
import { Colors } from '../constants/theme';

interface OnboardingOverviewScreenProps {
  onNext: () => void;
  onSkip: () => void;
}

export function OnboardingOverviewScreen({ onNext, onSkip }: OnboardingOverviewScreenProps) {
  return (
    <View style={styles.container}>
      <StatusBar style="dark" />
      
      {/* Header */}
      <View style={styles.header}>
        <View style={{ width: 40 }} />
        <TouchableOpacity onPress={onSkip} style={styles.skipButton}>
          <Text style={styles.skipText}>Skip</Text>
        </TouchableOpacity>
      </View>

      {/* Center Phone Mockup Section */}
      <View style={styles.heroContainer}>
        <View style={styles.phoneFrame}>
          <Image
            source={require('../../assets/onboarding_overview.png')}
            style={styles.mockupImage}
            resizeMode="cover"
          />
        </View>
      </View>

      {/* Bottom Sheet Card */}
      <View style={styles.bottomCard}>
        <Text style={styles.headline}>
          Get Onboard and Start{'\n'}
          <Text style={styles.highlightText}>Accepting Rides Instantly</Text>
        </Text>

        <Text style={styles.description}>
          Join India's fastest-growing delivery network. Start earning up to ₹1,500 daily with flexible shifts and instant payouts directly to your bank account.
        </Text>

        {/* Carousel Indicators & Controls */}
        <View style={styles.footerRow}>
          <View style={{ width: 44 }} />

          <View style={styles.indicators}>
            <View style={[styles.dot, styles.dotActive]} />
            <View style={styles.dot} />
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
    backgroundColor: '#f8f9ff',
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
  skipButton: {
    paddingVertical: 6,
    paddingHorizontal: 12,
  },
  skipText: {
    fontSize: 15,
    fontWeight: '600',
    color: Colors.primary,
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
    borderColor: '#ffffff',
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
    color: '#0b1c30',
    textAlign: 'center',
    lineHeight: 30,
    marginBottom: 12,
  },
  highlightText: {
    color: Colors.primary,
  },
  description: {
    fontSize: 13,
    color: '#5c647a',
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
    backgroundColor: '#e5eeff',
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
