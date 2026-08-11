import React, { useEffect, useRef } from 'react';
import { StyleSheet, Text, View, Image, TouchableOpacity, Animated, Dimensions } from 'react-native';
import { StatusBar } from 'expo-status-bar';
import { MaterialCommunityIcons } from '@expo/vector-icons';
import { Colors } from '../constants/theme';

interface GetStartedScreenProps {
  onGetStarted: () => void;
  onSignIn: () => void;
}

const { width } = Dimensions.get('window');

export function GetStartedScreen({ onGetStarted, onSignIn }: GetStartedScreenProps) {
  const floatAnim1 = useRef(new Animated.Value(0)).current;
  const floatAnim2 = useRef(new Animated.Value(0)).current;

  useEffect(() => {
    Animated.loop(
      Animated.sequence([
        Animated.timing(floatAnim1, {
          toValue: -12,
          duration: 2500,
          useNativeDriver: true,
        }),
        Animated.timing(floatAnim1, {
          toValue: 0,
          duration: 2500,
          useNativeDriver: true,
        }),
      ])
    ).start();

    Animated.loop(
      Animated.sequence([
        Animated.timing(floatAnim2, {
          toValue: -16,
          duration: 3000,
          useNativeDriver: true,
        }),
        Animated.timing(floatAnim2, {
          toValue: 0,
          duration: 3000,
          useNativeDriver: true,
        }),
      ])
    ).start();
  }, []);

  return (
    <View style={styles.container}>
      <StatusBar style="light" translucent backgroundColor="transparent" />
      
      {/* Top Hero Section */}
      <View style={styles.heroSection}>
        <Image
          source={require('../../assets/driver_hero.png')}
          style={styles.heroImage}
          resizeMode="cover"
        />
        <View style={styles.heroOverlay} />

        {/* Floating Icons from Stitch Spec */}
        <Animated.View style={[styles.floatingBadge, styles.badge1, { transform: [{ translateY: floatAnim1 }] }]}>
          <MaterialCommunityIcons name="map-marker" size={26} color="#ffffff" />
        </Animated.View>

        <Animated.View style={[styles.floatingBadge, styles.badge2, { transform: [{ translateY: floatAnim2 }] }]}>
          <MaterialCommunityIcons name="message-text" size={22} color="#ffffff" />
        </Animated.View>

        <Animated.View style={[styles.floatingBadge, styles.badge3, { transform: [{ translateY: floatAnim1 }] }]}>
          <MaterialCommunityIcons name="phone" size={22} color="#ffffff" />
        </Animated.View>

        <Animated.View style={[styles.floatingBadge, styles.badge4, { transform: [{ translateY: floatAnim2 }] }]}>
          <MaterialCommunityIcons name="account-circle" size={32} color="#ffffff" />
        </Animated.View>
      </View>

      {/* Bottom Content Sheet */}
      <View style={styles.contentSheet}>
        <View style={styles.sheetHandle} />

        <Text style={styles.headline}>Earn Money With This Driver App</Text>

        <Text style={styles.description}>
          Join the Avandab Transport Intelligence network. Stream live GPS telemetry, execute regional freight routes, and secure instant digital proof-of-delivery.
        </Text>

        <View style={styles.actionContainer}>
          <TouchableOpacity
            style={styles.primaryButton}
            activeOpacity={0.88}
            onPress={onGetStarted}
          >
            <Text style={styles.primaryButtonText}>Let's Get Started</Text>
          </TouchableOpacity>

          <TouchableOpacity
            style={styles.signInButton}
            activeOpacity={0.7}
            onPress={onSignIn}
          >
            <Text style={styles.signInText}>
              Already have an account? <Text style={styles.signInLink}>Sign In</Text>
            </Text>
          </TouchableOpacity>
        </View>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: Colors.background,
  },
  heroSection: {
    height: '52%',
    width: '100%',
    position: 'relative',
    backgroundColor: Colors.primary,
  },
  heroImage: {
    width: '100%',
    height: '100%',
  },
  heroOverlay: {
    ...StyleSheet.absoluteFillObject,
    backgroundColor: 'rgba(0, 40, 35, 0.25)',
  },
  floatingBadge: {
    position: 'absolute',
    backgroundColor: Colors.primary,
    borderRadius: 30,
    alignItems: 'center',
    justifyContent: 'center',
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 6 },
    shadowOpacity: 0.25,
    shadowRadius: 8,
    elevation: 8,
  },
  badge1: {
    top: '20%',
    left: '10%',
    width: 48,
    height: 48,
  },
  badge2: {
    top: '28%',
    right: '12%',
    width: 44,
    height: 44,
  },
  badge3: {
    bottom: '24%',
    left: '18%',
    width: 44,
    height: 44,
  },
  badge4: {
    bottom: '18%',
    right: '18%',
    width: 54,
    height: 54,
    borderWidth: 3,
    borderColor: '#ffffff',
  },
  contentSheet: {
    flex: 1,
    marginTop: -32,
    backgroundColor: '#ffffff',
    borderTopLeftRadius: 28,
    borderTopRightRadius: 28,
    paddingHorizontal: 24,
    paddingTop: 16,
    paddingBottom: 24,
    alignItems: 'center',
    shadowColor: '#000',
    shadowOffset: { width: 0, height: -4 },
    shadowOpacity: 0.1,
    shadowRadius: 12,
    elevation: 10,
  },
  sheetHandle: {
    width: 40,
    height: 4,
    borderRadius: 2,
    backgroundColor: '#e0e0e0',
    marginBottom: 20,
  },
  headline: {
    fontSize: 26,
    fontWeight: '700',
    color: '#0b1c30',
    textAlign: 'center',
    marginBottom: 12,
    lineHeight: 32,
  },
  description: {
    fontSize: 14,
    color: '#565e74',
    textAlign: 'center',
    lineHeight: 22,
    paddingHorizontal: 8,
    marginBottom: 24,
  },
  actionContainer: {
    width: '100%',
    alignItems: 'center',
    gap: 16,
  },
  primaryButton: {
    width: '100%',
    height: 54,
    borderRadius: 27,
    backgroundColor: Colors.primary,
    alignItems: 'center',
    justifyContent: 'center',
    shadowColor: Colors.primary,
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.3,
    shadowRadius: 8,
    elevation: 6,
  },
  primaryButtonText: {
    color: '#ffffff',
    fontSize: 17,
    fontWeight: '600',
  },
  signInButton: {
    paddingVertical: 8,
  },
  signInText: {
    fontSize: 14,
    color: '#565e74',
  },
  signInLink: {
    color: Colors.primary,
    fontWeight: '700',
    textDecorationLine: 'underline',
  },
});
