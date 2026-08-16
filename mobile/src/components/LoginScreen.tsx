import React, { useState } from 'react';
import { StyleSheet, Text, View, TextInput, TouchableOpacity, ActivityIndicator, Alert } from 'react-native';
import { StatusBar } from 'expo-status-bar';
import { MaterialCommunityIcons } from '@expo/vector-icons';
import { Colors } from '../constants/theme';
import { getApiBaseURL } from '../constants/network';
import { useAuthStore } from '../stores/authStore';

interface LoginScreenProps {
  onLoginSuccess: () => void;
  onForgotPassword?: () => void;
  onRegisterLink?: () => void;
}

export function LoginScreen({ onLoginSuccess, onForgotPassword, onRegisterLink }: LoginScreenProps) {
  const [email, setEmail] = useState('driver@avandab.com');
  const [password, setPassword] = useState('password123');
  const [showPassword, setShowPassword] = useState(false);
  const [loading, setLoading] = useState(false);

  const setAuth = useAuthStore((state) => state.setAuth);

  const handleSignIn = async () => {
    if (!email || !password) {
      Alert.alert('Missing Fields', 'Please enter both email and password.');
      return;
    }

    setLoading(true);

    try {
      const targetUrl = `${getApiBaseURL()}/api/v1/auth/token`;

      const response = await fetch(targetUrl, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password }),
      });

      if (!response.ok) {
        const errText = await response.text();
        setLoading(false);
        Alert.alert('Sign In Failed', errText || `Server returned HTTP ${response.status}.`);
        return;
      }

      const data = await response.json();

      if (!data.token) {
        setLoading(false);
        Alert.alert('Sign In Failed', 'Server response did not include an authentication token.');
        return;
      }

      await setAuth(data.token, {
        id: data.user?.id || '',
        name: data.user?.name || '',
        role: 'DRIVER',
        email: email,
      });
      setLoading(false);
      onLoginSuccess();
    } catch (err: any) {
      setLoading(false);
      Alert.alert('Sign In Failed', err?.message || 'Unable to reach the server. Please try again.');
    }
  };

  return (
    <View style={styles.container}>
      <StatusBar style="dark" />

      {/* Main Login Card */}
      <View style={styles.card}>
        {/* Header */}
        <View style={styles.header}>
          <Text style={styles.brandTitle}>Avandab</Text>
          <Text style={styles.brandSubtitle}>Logistics Management Portal</Text>
        </View>

        {/* Input Fields */}
        <View style={styles.formGroup}>
          <Text style={styles.label}>EMAIL ADDRESS</Text>
          <View style={styles.inputWrapper}>
            <MaterialCommunityIcons name="email-outline" size={20} color="#6d7a77" style={styles.inputIcon} />
            <TextInput
              style={styles.input}
              placeholder="dispatcher@fleet.com"
              placeholderTextColor="#bcc9c6"
              value={email}
              onChangeText={setEmail}
              keyboardType="email-address"
              autoCapitalize="none"
            />
          </View>
        </View>

        <View style={styles.formGroup}>
          <View style={styles.labelRow}>
            <Text style={styles.label}>PASSWORD</Text>
            {onForgotPassword && (
              <TouchableOpacity onPress={onForgotPassword}>
                <Text style={styles.forgotText}>Forgot Password?</Text>
              </TouchableOpacity>
            )}
          </View>
          <View style={styles.inputWrapper}>
            <MaterialCommunityIcons name="lock-outline" size={20} color="#6d7a77" style={styles.inputIcon} />
            <TextInput
              style={[styles.input, { paddingRight: 40 }]}
              placeholder="••••••••"
              placeholderTextColor="#bcc9c6"
              value={password}
              onChangeText={setPassword}
              secureTextEntry={!showPassword}
            />
            <TouchableOpacity style={styles.eyeIcon} onPress={() => setShowPassword(!showPassword)}>
              <MaterialCommunityIcons
                name={showPassword ? 'eye-off-outline' : 'eye-outline'}
                size={20}
                color="#6d7a77"
              />
            </TouchableOpacity>
          </View>
        </View>

        {/* Sign In Button */}
        <TouchableOpacity
          style={styles.submitBtn}
          activeOpacity={0.88}
          onPress={handleSignIn}
          disabled={loading}
        >
          {loading ? (
            <ActivityIndicator color="#ffffff" />
          ) : (
            <View style={styles.btnContent}>
              <Text style={styles.submitBtnText}>Sign In</Text>
              <MaterialCommunityIcons name="arrow-right" size={18} color="#ffffff" />
            </View>
          )}
        </TouchableOpacity>

        {/* Divider */}
        <View style={styles.dividerRow}>
          <View style={styles.dividerLine} />
          <Text style={styles.dividerText}>OR CONTINUE WITH</Text>
          <View style={styles.dividerLine} />
        </View>

        {/* OAuth Social Buttons */}
        <View style={styles.socialRow}>
          <TouchableOpacity style={styles.socialBtn} activeOpacity={0.8} onPress={handleSignIn}>
            <MaterialCommunityIcons name="google" size={20} color="#4285F4" />
            <Text style={styles.socialBtnText}>Google</Text>
          </TouchableOpacity>

          <TouchableOpacity style={styles.socialBtn} activeOpacity={0.8} onPress={handleSignIn}>
            <MaterialCommunityIcons name="microsoft" size={20} color="#00a4ef" />
            <Text style={styles.socialBtnText}>Microsoft</Text>
          </TouchableOpacity>
        </View>

        {/* Register Link */}
        {onRegisterLink && (
          <TouchableOpacity style={{ marginTop: 20, alignItems: 'center' }} onPress={onRegisterLink}>
            <Text style={{ fontSize: 13, color: '#565e74' }}>
              Don't have a driver account? <Text style={{ color: Colors.primary, fontWeight: '700' }}>Register Now</Text>
            </Text>
          </TouchableOpacity>
        )}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#f7f9fb',
    justifyContent: 'center',
    paddingHorizontal: 24,
  },
  card: {
    backgroundColor: '#ffffff',
    borderRadius: 16,
    padding: 28,
    borderWidth: 1,
    borderColor: '#e2e8f0',
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.04,
    shadowRadius: 12,
    elevation: 4,
  },
  header: {
    alignItems: 'center',
    marginBottom: 28,
  },
  brandTitle: {
    fontSize: 34,
    fontWeight: '700',
    color: Colors.primary,
    letterSpacing: -0.5,
    marginBottom: 4,
  },
  brandSubtitle: {
    fontSize: 14,
    color: '#3d4947',
    fontWeight: '400',
  },
  formGroup: {
    marginBottom: 20,
  },
  labelRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 8,
  },
  label: {
    fontSize: 11,
    fontWeight: '700',
    color: '#3d4947',
    letterSpacing: 0.5,
    marginBottom: 8,
  },
  forgotText: {
    fontSize: 13,
    color: Colors.primary,
    fontWeight: '500',
  },
  inputWrapper: {
    position: 'relative',
    justifyContent: 'center',
  },
  inputIcon: {
    position: 'absolute',
    left: 12,
    zIndex: 10,
  },
  input: {
    height: 48,
    backgroundColor: '#ffffff',
    borderWidth: 1,
    borderColor: '#bcc9c6',
    borderRadius: 8,
    paddingLeft: 40,
    paddingRight: 14,
    fontSize: 14,
    color: '#191c1e',
  },
  eyeIcon: {
    position: 'absolute',
    right: 12,
    padding: 4,
  },
  submitBtn: {
    height: 48,
    backgroundColor: Colors.primary,
    borderRadius: 8,
    alignItems: 'center',
    justifyContent: 'center',
    marginTop: 8,
    marginBottom: 24,
    shadowColor: Colors.primary,
    shadowOffset: { width: 0, height: 3 },
    shadowOpacity: 0.2,
    shadowRadius: 6,
    elevation: 4,
  },
  btnContent: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
  },
  submitBtnText: {
    color: '#ffffff',
    fontSize: 15,
    fontWeight: '600',
  },
  dividerRow: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 20,
  },
  dividerLine: {
    flex: 1,
    height: 1,
    backgroundColor: '#e2e8f0',
  },
  dividerText: {
    fontSize: 10,
    fontWeight: '700',
    color: '#6d7a77',
    paddingHorizontal: 12,
    letterSpacing: 0.5,
  },
  socialRow: {
    flexDirection: 'row',
    gap: 12,
  },
  socialBtn: {
    flex: 1,
    height: 44,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: '#bcc9c6',
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 8,
    backgroundColor: '#ffffff',
  },
  socialBtnText: {
    fontSize: 14,
    fontWeight: '600',
    color: '#191c1e',
  },
});
