import React, { useState } from 'react';
import { StyleSheet, Text, View, TextInput, TouchableOpacity, ScrollView, ActivityIndicator, Alert } from 'react-native';
import { StatusBar } from 'expo-status-bar';
import { MaterialCommunityIcons } from '@expo/vector-icons';
import { Colors } from '../constants/theme';
import { getApiBaseURL } from '../constants/network';
import { useAuthStore } from '../stores/authStore';

interface RegisterScreenProps {
  onRegisterSuccess: () => void;
  onBackToLogin: () => void;
}

export function RegisterScreen({ onRegisterSuccess, onBackToLogin }: RegisterScreenProps) {
  const [fullName, setFullName] = useState('');
  const [email, setEmail] = useState('');
  const [phone, setPhone] = useState('');
  const [password, setPassword] = useState('');
  const [vehicleNumber, setVehicleNumber] = useState('');
  const [loading, setLoading] = useState(false);

  const setAuth = useAuthStore((state) => state.setAuth);

  const handleRegister = async () => {
    if (!fullName || !email || !password) {
      Alert.alert('Missing Fields', 'Please fill in all required fields.');
      return;
    }

    setLoading(true);

    try {
      const targetUrl = `${getApiBaseURL()}/api/v1/auth/register`;
      const response = await fetch(targetUrl, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: fullName,
          email,
          phone,
          password,
          vehicle_number: vehicleNumber || 'MH-12-AB-9942',
        }),
      });

      if (!response.ok) {
        const errText = await response.text();
        setLoading(false);
        Alert.alert('Registration Failed', errText || `Server returned HTTP ${response.status}.`);
        return;
      }

      const data = await response.json();

      if (!data.token) {
        setLoading(false);
        Alert.alert('Registration Failed', 'Server response did not include an authentication token.');
        return;
      }

      await setAuth(data.token, {
        id: data.user?.id || '',
        name: fullName,
        role: 'DRIVER',
        email,
      });
      setLoading(false);
      onRegisterSuccess();
    } catch (err: any) {
      setLoading(false);
      Alert.alert('Registration Failed', err?.message || 'Unable to reach the server. Please try again.');
    }
  };

  return (
    <View style={styles.container}>
      <StatusBar style="dark" />
      
      {/* Top Header */}
      <View style={styles.header}>
        <TouchableOpacity style={styles.iconButton} onPress={onBackToLogin}>
          <MaterialCommunityIcons name="arrow-left" size={22} color="#0b1c30" />
        </TouchableOpacity>
        <Text style={styles.headerTitle}>Driver Account Registration</Text>
        <TouchableOpacity onPress={onBackToLogin}>
          <Text style={styles.skipText}>CANCEL</Text>
        </TouchableOpacity>
      </View>

      <ScrollView contentContainerStyle={styles.scrollContent} showsVerticalScrollIndicator={false}>
        <View style={styles.titleSection}>
          <Text style={styles.title}>Driver Profile</Text>
          <Text style={styles.subtitle}>Please provide your details to start accepting rides.</Text>
        </View>

        {/* Inputs */}
        <View style={styles.formGroup}>
          <Text style={styles.label}>FULL NAME</Text>
          <View style={styles.inputWrapper}>
            <MaterialCommunityIcons name="account-outline" size={20} color="#6d7a77" style={styles.inputIcon} />
            <TextInput
              style={styles.input}
              placeholder="e.g. Rajesh Kumar"
              placeholderTextColor="#bcc9c6"
              value={fullName}
              onChangeText={setFullName}
            />
          </View>
        </View>

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

        <View style={styles.formGroup}>
          <Text style={styles.label}>PHONE NUMBER</Text>
          <View style={styles.inputWrapper}>
            <MaterialCommunityIcons name="phone-outline" size={20} color="#6d7a77" style={styles.inputIcon} />
            <TextInput
              style={styles.input}
              placeholder="+91 98765 43210"
              placeholderTextColor="#bcc9c6"
              value={phone}
              onChangeText={setPhone}
              keyboardType="phone-pad"
            />
          </View>
        </View>

        <View style={styles.formGroup}>
          <Text style={styles.label}>VEHICLE REGISTRATION</Text>
          <View style={styles.inputWrapper}>
            <MaterialCommunityIcons name="truck-outline" size={20} color="#6d7a77" style={styles.inputIcon} />
            <TextInput
              style={styles.input}
              placeholder="e.g. MH-12-AB-9942"
              placeholderTextColor="#bcc9c6"
              value={vehicleNumber}
              onChangeText={setVehicleNumber}
              autoCapitalize="characters"
            />
          </View>
        </View>

        <View style={styles.formGroup}>
          <Text style={styles.label}>PASSWORD</Text>
          <View style={styles.inputWrapper}>
            <MaterialCommunityIcons name="lock-outline" size={20} color="#6d7a77" style={styles.inputIcon} />
            <TextInput
              style={styles.input}
              placeholder="••••••••"
              placeholderTextColor="#bcc9c6"
              value={password}
              onChangeText={setPassword}
              secureTextEntry
            />
          </View>
        </View>

        {/* Submit Button */}
        <TouchableOpacity
          style={styles.submitBtn}
          activeOpacity={0.88}
          onPress={handleRegister}
          disabled={loading}
        >
          {loading ? (
            <ActivityIndicator color="#ffffff" />
          ) : (
            <View style={styles.btnContent}>
              <Text style={styles.submitBtnText}>Complete Registration</Text>
              <MaterialCommunityIcons name="arrow-right" size={18} color="#ffffff" />
            </View>
          )}
        </TouchableOpacity>

        <TouchableOpacity style={styles.loginLink} onPress={onBackToLogin}>
          <Text style={styles.loginLinkText}>
            Already registered? <Text style={styles.loginLinkHighlight}>Sign In</Text>
          </Text>
        </TouchableOpacity>
      </ScrollView>
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
    paddingHorizontal: 20,
    paddingTop: 50,
    paddingBottom: 16,
  },
  headerTitle: {
    fontSize: 16,
    fontWeight: '600',
    color: '#0b1c30',
  },
  iconButton: {
    width: 38,
    height: 38,
    borderRadius: 12,
    backgroundColor: '#eff4ff',
    alignItems: 'center',
    justifyContent: 'center',
  },
  skipText: {
    fontSize: 12,
    fontWeight: '700',
    color: Colors.primary,
    letterSpacing: 1,
  },
  progressContainer: {
    paddingHorizontal: 20,
    marginBottom: 16,
  },
  progressBarBackground: {
    height: 4,
    backgroundColor: '#d3e4fe',
    borderRadius: 2,
    overflow: 'hidden',
  },
  progressBarFill: {
    width: '33%',
    height: '100%',
    backgroundColor: Colors.primary,
  },
  stepText: {
    fontSize: 11,
    color: '#3d4947',
    textAlign: 'right',
    marginTop: 4,
    fontWeight: '500',
  },
  scrollContent: {
    paddingHorizontal: 20,
    paddingBottom: 40,
  },
  titleSection: {
    marginBottom: 20,
  },
  title: {
    fontSize: 26,
    fontWeight: '700',
    color: '#0b1c30',
    marginBottom: 4,
  },
  subtitle: {
    fontSize: 14,
    color: '#3d4947',
    lineHeight: 20,
  },
  formGroup: {
    marginBottom: 16,
  },
  label: {
    fontSize: 11,
    fontWeight: '700',
    color: '#3d4947',
    letterSpacing: 0.5,
    marginBottom: 6,
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
    height: 50,
    backgroundColor: Colors.primary,
    borderRadius: 8,
    alignItems: 'center',
    justifyContent: 'center',
    marginTop: 12,
    marginBottom: 16,
    shadowColor: Colors.primary,
    shadowOffset: { width: 0, height: 3 },
    shadowOpacity: 0.25,
    shadowRadius: 6,
    elevation: 4,
  },
  btnContent: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
  },
  submitBtnText: {
    color: '#ffffff',
    fontSize: 15,
    fontWeight: '600',
  },
  loginLink: {
    alignItems: 'center',
    paddingVertical: 8,
  },
  loginLinkText: {
    fontSize: 14,
    color: '#565e74',
  },
  loginLinkHighlight: {
    color: Colors.primary,
    fontWeight: '700',
    textDecorationLine: 'underline',
  },
});
