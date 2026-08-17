import React, { useEffect } from 'react';
import { StyleSheet, Text, View, Animated } from 'react-native';
import { StatusBar } from 'expo-status-bar';
import { MaterialCommunityIcons } from '@expo/vector-icons';
import { Colors, Font, Radius } from '../constants/theme';

interface SplashScreenProps {
  onFinish: () => void;
}

export function SplashScreen({ onFinish }: SplashScreenProps) {
  const logoOpacity = new Animated.Value(0);
  const logoTranslateY = new Animated.Value(10);
  const textOpacity = new Animated.Value(0);
  const textTranslateY = new Animated.Value(10);

  useEffect(() => {
    Animated.sequence([
      Animated.parallel([
        Animated.timing(logoOpacity, {
          toValue: 1,
          duration: 600,
          useNativeDriver: true,
        }),
        Animated.timing(logoTranslateY, {
          toValue: 0,
          duration: 600,
          useNativeDriver: true,
        }),
      ]),
      Animated.parallel([
        Animated.timing(textOpacity, {
          toValue: 1,
          duration: 500,
          useNativeDriver: true,
        }),
        Animated.timing(textTranslateY, {
          toValue: 0,
          duration: 500,
          useNativeDriver: true,
        }),
      ]),
    ]).start();

    const timer = setTimeout(() => {
      onFinish();
    }, 2000);

    return () => clearTimeout(timer);
  }, []);

  return (
    <View style={styles.container}>
      <StatusBar style="light" backgroundColor={Colors.chrome} />

      <View style={styles.content}>
        <Animated.View
          style={[
            styles.logoContainer,
            {
              opacity: logoOpacity,
              transform: [{ translateY: logoTranslateY }],
            },
          ]}
        >
          <MaterialCommunityIcons name="truck-fast" size={56} color={Colors.primary} />
        </Animated.View>

        <Animated.View
          style={[
            styles.textContainer,
            {
              opacity: textOpacity,
              transform: [{ translateY: textTranslateY }],
            },
          ]}
        >
          <Text style={styles.brandTitle}>AVANDAB</Text>
          <View style={styles.divider} />
          <Text style={styles.brandSubtitle}>DRIVER OPS</Text>
        </Animated.View>

        <Text style={styles.versionTag}>v2.4.1 · FLEET MOBILE</Text>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: Colors.chrome,
    justifyContent: 'center',
    alignItems: 'center',
  },
  content: {
    alignItems: 'center',
    justifyContent: 'center',
  },
  logoContainer: {
    width: 96,
    height: 96,
    borderRadius: Radius.lg,
    backgroundColor: Colors.chromeLight,
    borderWidth: 1,
    borderColor: Colors.chromeBorder,
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: 20,
  },
  textContainer: {
    alignItems: 'center',
  },
  brandTitle: {
    color: Colors.textOnChrome,
    fontSize: 28,
    fontWeight: '900',
    letterSpacing: 4,
    fontFamily: Font.mono,
  },
  divider: {
    width: 40,
    height: 1,
    backgroundColor: Colors.primary,
    marginVertical: 8,
  },
  brandSubtitle: {
    color: Colors.primary,
    fontSize: 12,
    fontWeight: '700',
    letterSpacing: 3,
    fontFamily: Font.mono,
  },
  versionTag: {
    color: Colors.textOnChromeMuted,
    fontSize: 10,
    fontWeight: '600',
    letterSpacing: 1,
    fontFamily: Font.mono,
    marginTop: 32,
  },
});
