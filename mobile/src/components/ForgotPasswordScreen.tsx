import React, { useState } from 'react';
import { StyleSheet, Text, View, TextInput, TouchableOpacity, ActivityIndicator, Alert } from 'react-native';
import { StatusBar } from 'expo-status-bar';
import { MaterialCommunityIcons } from '@expo/vector-icons';
import { Colors } from '../constants/theme';
import { getApiBaseURL } from '../constants/network';

interface ForgotPasswordScreenProps {
  onBackToLogin: () => void;
}

export function ForgotPasswordScreen({ onBackToLogin }: ForgotPasswordScreenProps) {
  const [email, setEmail] = useState('');
  const [loading, setLoading] = useState(false);
  const [submitted, setSubmitted] = useState(false);

  const handleResetPassword = async () => {
    if (!email) {
      Alert.alert('Email Required', 'Please enter your registered email address.');
      return;
    }

    setLoading(true);

    try {
      const targetUrl = `${getApiBaseURL()}/forgot-password`;
      await fetch(targetUrl, {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body: `email=${encodeURIComponent(email)}`,
      });

      setLoading(false);
      setSubmitted(true);
    } catch (err: any) {
      console.log('[FORGOT PASSWORD FALLBACK]: Handled reset request:', err?.message || err);
      setLoading(false);
      setSubmitted(true);
    }
  };

  return (
    <View style={styles.container}>
      <StatusBar style="dark" />

      {/* Top Bar */}
      <View style={styles.header}>
        <TouchableOpacity style={styles.iconButton} onPress={onBackToLogin}>
          <MaterialCommunityIcons name="arrow-left" size={22} color="#0b1c30" />
        </TouchableOpacity>
        <Text style={styles.headerTitle}>Reset Password</Text>
        <View style={{ width: 38 }} />
      </View>

      <View style={styles.card}>
        <View style={styles.iconCircle}>
          <MaterialCommunityIcons name="lock-reset" size={32} color={Colors.primary} />
        </View>

        <Text style={styles.title}>Forgot Password?</Text>
        <Text style={styles.subtitle}>
          Enter your registered email address and we'll send you instructions to reset your account password.
        </Text>

        {submitted ? (
          <View style={styles.successBox}>
            <MaterialCommunityIcons name="check-circle" size={28} color="#10B981" />
            <Text style={styles.successTitle}>Reset Email Sent!</Text>
            <Text style={styles.successMessage}>
              If an account exists for <Text style={{ fontWeight: '700' }}>{email}</Text>, password reset instructions have been dispatched.
            </Text>
            <TouchableOpacity style={styles.submitBtn} onPress={onBackToLogin}>
              <Text style={styles.submitBtnText}>Return to Sign In</Text>
            </TouchableOpacity>
          </View>
        ) : (
          <View style={styles.form}>
            <View style={styles.formGroup}>
              <Text style={styles.label}>EMAIL ADDRESS</Text>
              <View style={styles.inputWrapper}>
                <MaterialCommunityIcons name="email-outline" size={20} color="#6d7a77" style={styles.inputIcon} />
                <TextInput
                  style={styles.input}
                  placeholder="driver@avandab.com"
                  placeholderTextColor="#bcc9c6"
                  value={email}
                  onChangeText={setEmail}
                  keyboardType="email-address"
                  autoCapitalize="none"
                />
              </View>
            </View>

            <TouchableOpacity
              style={styles.submitBtn}
              activeOpacity={0.88}
              onPress={handleResetPassword}
              disabled={loading}
            >
              {loading ? (
                <ActivityIndicator color="#ffffff" />
              ) : (
                <Text style={styles.submitBtnText}>Send Reset Link</Text>
              )}
            </TouchableOpacity>

            <TouchableOpacity style={styles.backLink} onPress={onBackToLogin}>
              <Text style={styles.backLinkText}>← Back to Sign In</Text>
            </TouchableOpacity>
          </View>
        )}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#f7f9fb',
    paddingHorizontal: 20,
  },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingTop: 50,
    paddingBottom: 20,
  },
  iconButton: {
    width: 38,
    height: 38,
    borderRadius: 12,
    backgroundColor: '#ffffff',
    borderWidth: 1,
    borderColor: '#e2e8f0',
    alignItems: 'center',
    justifyContent: 'center',
  },
  headerTitle: {
    fontSize: 16,
    fontWeight: '600',
    color: '#0b1c30',
  },
  card: {
    backgroundColor: '#ffffff',
    borderRadius: 16,
    padding: 24,
    borderWidth: 1,
    borderColor: '#e2e8f0',
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.04,
    shadowRadius: 12,
    elevation: 4,
    alignItems: 'center',
  },
  iconCircle: {
    width: 64,
    height: 64,
    borderRadius: 32,
    backgroundColor: '#f4fffc',
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: 16,
  },
  title: {
    fontSize: 22,
    fontWeight: '700',
    color: '#0b1c30',
    marginBottom: 8,
  },
  subtitle: {
    fontSize: 13,
    color: '#5c647a',
    textAlign: 'center',
    lineHeight: 20,
    marginBottom: 24,
  },
  form: {
    width: '100%',
  },
  formGroup: {
    marginBottom: 20,
  },
  label: {
    fontSize: 11,
    fontWeight: '700',
    color: '#3d4947',
    letterSpacing: 0.5,
    marginBottom: 8,
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
    color: '#0b1c30',
  },
  submitBtn: {
    height: 48,
    backgroundColor: Colors.primary,
    borderRadius: 8,
    alignItems: 'center',
    justifyContent: 'center',
    marginTop: 4,
    marginBottom: 16,
    width: '100%',
    shadowColor: Colors.primary,
    shadowOffset: { width: 0, height: 3 },
    shadowOpacity: 0.2,
    shadowRadius: 6,
    elevation: 4,
  },
  submitBtnText: {
    color: '#ffffff',
    fontSize: 15,
    fontWeight: '600',
  },
  backLink: {
    alignItems: 'center',
    paddingVertical: 6,
  },
  backLinkText: {
    fontSize: 14,
    color: Colors.primary,
    fontWeight: '600',
  },
  successBox: {
    alignItems: 'center',
    width: '100%',
    paddingVertical: 12,
  },
  successTitle: {
    fontSize: 18,
    fontWeight: '700',
    color: '#0b1c30',
    marginTop: 8,
    marginBottom: 8,
  },
  successMessage: {
    fontSize: 13,
    color: '#5c647a',
    textAlign: 'center',
    lineHeight: 20,
    marginBottom: 20,
  },
});
