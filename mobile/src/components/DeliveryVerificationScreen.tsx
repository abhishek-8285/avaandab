import React, { useState } from 'react';
import {
  StyleSheet,
  Text,
  View,
  TouchableOpacity,
  ScrollView,
  Image,
  TextInput,
  ActivityIndicator,
  Alert,
} from 'react-native';
import { StatusBar } from 'expo-status-bar';
import { MaterialCommunityIcons } from '@expo/vector-icons';
import { CameraView, useCameraPermissions } from 'expo-camera';
import * as ImageManipulator from 'expo-image-manipulator';
import { Colors, Font, Radius, Spacing } from '../constants/theme';
import { getApiBaseURL } from '../constants/network';
import { useAuthStore } from '../stores/authStore';
import { OfflineQueue } from '../services/offlineQueue';

interface DeliveryVerificationScreenProps {
  tripId?: string;
  onComplete: () => void;
  onBack: () => void;
}

export function DeliveryVerificationScreen({
  tripId = '1',
  onComplete,
  onBack,
}: DeliveryVerificationScreenProps) {
  const [permission, requestPermission] = useCameraPermissions();
  const [cameraActive, setCameraActive] = useState(false);
  const [capturedPhoto, setCapturedPhoto] = useState<string | null>(null);
  const [cameraRef, setCameraRef] = useState<any>(null);

  const [consigneeName, setConsigneeName] = useState('');
  const [consigneePhone, setConsigneePhone] = useState('');
  const [notes, setNotes] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const compressPhoto = async (uri: string): Promise<string> => {
    try {
      const info = await ImageManipulator.manipulateAsync(uri, [], {
        compress: 0.7,
        format: ImageManipulator.SaveFormat.JPEG,
      });
      return info.uri;
    } catch {
      return uri;
    }
  };

  const takePhoto = async () => {
    if (cameraRef) {
      try {
        const photo = await cameraRef.takePictureAsync();
        const compressedUri = await compressPhoto(photo.uri);
        setCapturedPhoto(compressedUri);
        setCameraActive(false);
      } catch {
        Alert.alert('Camera Error', 'Failed to capture photo proof.');
      }
    }
  };

  const submit = async () => {
    if (!capturedPhoto && !consigneeName.trim()) {
      Alert.alert('Missing Fields', 'Please add a consignee name or capture a photo proof.');
      return;
    }

    setSubmitting(true);
    const form = new FormData();
    form.append('consignee_name', consigneeName.trim());
    if (consigneePhone.trim()) {
      form.append('consignee_phone', consigneePhone.trim());
    }
    if (notes.trim()) {
      form.append('notes', notes.trim());
    }
    if (capturedPhoto) {
      form.append('pod_photo', {
        uri: capturedPhoto,
        name: 'pod.jpg',
        type: 'image/jpeg',
      } as any);
    }

    try {
      const token = useAuthStore.getState().token;
      const targetUrl = `${getApiBaseURL()}/api/v1/trips/${tripId}/deliver-pod`;

      const res = await fetch(targetUrl, {
        method: 'POST',
        headers: {
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: form,
      });

      if (!res.ok) {
        const errBody = await res.json().catch(() => ({}));
        throw new Error(errBody.error || `HTTP ${res.status}`);
      }

      const json = await res.json();
      await OfflineQueue.clearPOD(tripId);
      Alert.alert('Delivered', `Trip ${json.trip_number || tripId} marked as delivered successfully!`, [
        { text: 'OK', onPress: onComplete },
      ]);
    } catch {
      // Queue for automatic offline sync retry
      await OfflineQueue.enqueuePOD(tripId, {
        consignee_name: consigneeName.trim() || 'Consignee',
        notes: notes.trim(),
        photo_uri: capturedPhoto,
      });
      Alert.alert('Saved Offline', 'Delivery proof queued in offline storage. Will submit when back online.', [
        { text: 'OK', onPress: onComplete },
      ]);
    } finally {
      setSubmitting(false);
    }
  };

  const isSubmitDisabled = submitting || (!capturedPhoto && !consigneeName.trim());

  return (
    <View style={styles.container}>
      <StatusBar style="light" />

      <View style={styles.header}>
        <TouchableOpacity style={styles.iconButton} onPress={onBack}>
          <MaterialCommunityIcons name="arrow-left" size={18} color={Colors.textOnChrome} />
        </TouchableOpacity>
        <Text style={styles.headerLabel}>POD VERIFICATION</Text>
        <TouchableOpacity style={styles.iconButton}>
          <MaterialCommunityIcons name="bell-outline" size={14} color={Colors.textOnChrome} />
        </TouchableOpacity>
      </View>

      {cameraActive ? (
        <View style={styles.cameraContainer}>
          {!permission?.granted ? (
            <View style={styles.permissionBox}>
              <Text style={styles.permissionText}>Camera permission required to capture proof of delivery.</Text>
              <TouchableOpacity style={styles.permissionBtn} onPress={requestPermission}>
                <Text style={styles.permissionBtnText}>GRANT CAMERA</Text>
              </TouchableOpacity>
              <TouchableOpacity style={styles.cancelCameraBtn} onPress={() => setCameraActive(false)}>
                <Text style={styles.cancelCameraText}>CANCEL</Text>
              </TouchableOpacity>
            </View>
          ) : (
            <CameraView style={styles.cameraView} ref={(ref) => setCameraRef(ref)}>
              <View style={styles.cameraOverlay}>
                <View style={styles.scannerFrame} />
                <Text style={styles.cameraGuideText}>ALIGN BARCODE / CARGO IN FRAME</Text>
                <TouchableOpacity style={styles.captureBtn} onPress={takePhoto}>
                  <View style={styles.captureInnerCircle} />
                </TouchableOpacity>
                <TouchableOpacity style={styles.closeCameraBtn} onPress={() => setCameraActive(false)}>
                  <MaterialCommunityIcons name="close" size={20} color={Colors.textOnChrome} />
                </TouchableOpacity>
              </View>
            </CameraView>
          )}
        </View>
      ) : (
        <ScrollView contentContainerStyle={styles.scrollContent} showsVerticalScrollIndicator={false}>
          <View style={styles.titleSection}>
            <Text style={styles.title}>COMPLETE DELIVERY</Text>
            <View style={styles.titleUnderline} />
            <Text style={styles.subtitle}>TRIP REF · #{tripId}</Text>
          </View>

          {/* Consignee Details Input Form */}
          <View style={styles.card}>
            <View style={styles.cardHeaderRow}>
              <Text style={styles.cardHeader}>CONSIGNEE DETAILS</Text>
              <Text style={styles.cardMeta}>REQUIRED</Text>
            </View>

            <View style={styles.formGroup}>
              <Text style={styles.label}>RECEIVER / CONSIGNEE NAME</Text>
              <TextInput
                style={styles.input}
                placeholder="e.g. Rajesh Sharma"
                placeholderTextColor={Colors.textMuted}
                value={consigneeName}
                onChangeText={setConsigneeName}
              />
            </View>

            <View style={styles.formGroup}>
              <Text style={styles.label}>PHONE NUMBER (OPTIONAL)</Text>
              <TextInput
                style={styles.input}
                placeholder="+91 98765 43210"
                placeholderTextColor={Colors.textMuted}
                value={consigneePhone}
                onChangeText={setConsigneePhone}
                keyboardType="phone-pad"
              />
            </View>

            <View style={styles.formGroup}>
              <Text style={styles.label}>DELIVERY REMARKS / NOTES</Text>
              <TextInput
                style={[styles.input, styles.textArea]}
                placeholder="e.g. Received at Gate 3 with intact seal"
                placeholderTextColor={Colors.textMuted}
                value={notes}
                onChangeText={setNotes}
                multiline
                numberOfLines={3}
              />
            </View>
          </View>

          {/* Photo POD Card */}
          <View style={styles.card}>
            <View style={styles.cardHeaderRow}>
              <Text style={styles.cardHeader}>PHOTO PROOF</Text>
              {capturedPhoto ? (
                <Text style={styles.cardMetaSuccess}>CAPTURED</Text>
              ) : (
                <Text style={styles.cardMeta}>RECOMMENDED</Text>
              )}
            </View>
            <Text style={styles.cardSubtitle}>
              Capture clear photo of delivered cargo, invoice, or receiver sign-off.
            </Text>

            {capturedPhoto ? (
              <View style={styles.photoPreviewContainer}>
                <Image source={{ uri: capturedPhoto }} style={styles.photoPreview} />
                <TouchableOpacity style={styles.retakeBtn} onPress={() => setCameraActive(true)}>
                  <MaterialCommunityIcons name="camera-retake-outline" size={14} color={Colors.textOnPrimary} />
                  <Text style={styles.retakeBtnText}>RETAKE</Text>
                </TouchableOpacity>
              </View>
            ) : (
              <TouchableOpacity
                style={styles.photoPlaceholder}
                activeOpacity={0.8}
                onPress={() => {
                  if (!permission?.granted) {
                    requestPermission().then((res) => {
                      if (res.granted) setCameraActive(true);
                    });
                  } else {
                    setCameraActive(true);
                  }
                }}
              >
                <MaterialCommunityIcons name="camera-plus-outline" size={28} color={Colors.primary} />
                <Text style={styles.placeholderTitle}>TAP TO CAPTURE</Text>
                <Text style={styles.placeholderSub}>Barcode, cargo or POD stamp</Text>
              </TouchableOpacity>
            )}
          </View>

          {/* Submit Action Button */}
          <TouchableOpacity
            style={[styles.submitBtn, isSubmitDisabled && styles.submitBtnDisabled]}
            activeOpacity={0.88}
            onPress={submit}
            disabled={isSubmitDisabled}
          >
            {submitting ? (
              <ActivityIndicator color={Colors.textOnPrimary} size="small" />
            ) : (
              <>
                <Text style={styles.submitBtnText}>CONFIRM & SUBMIT E-POD</Text>
                <MaterialCommunityIcons name="check-circle-outline" size={16} color={Colors.textOnPrimary} />
              </>
            )}
          </TouchableOpacity>
        </ScrollView>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: Colors.background,
  },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingHorizontal: Spacing.lg,
    paddingTop: 50,
    paddingBottom: Spacing.md,
    backgroundColor: Colors.chrome,
  },
  headerLabel: {
    fontSize: 11,
    fontWeight: '700',
    color: Colors.textOnChrome,
    letterSpacing: 2,
    fontFamily: Font.mono,
  },
  iconButton: {
    width: 32,
    height: 32,
    borderRadius: Radius.md,
    borderWidth: 1,
    borderColor: Colors.chromeBorder,
    alignItems: 'center',
    justifyContent: 'center',
  },
  scrollContent: {
    paddingHorizontal: Spacing.lg,
    paddingTop: Spacing.lg,
    paddingBottom: 40,
  },
  titleSection: {
    marginBottom: Spacing.lg,
  },
  title: {
    fontSize: 18,
    fontWeight: '900',
    color: Colors.textPrimary,
    letterSpacing: 2,
    fontFamily: Font.mono,
  },
  titleUnderline: {
    width: 28,
    height: 2,
    backgroundColor: Colors.primary,
    marginTop: 6,
    marginBottom: 8,
  },
  subtitle: {
    fontSize: 11,
    color: Colors.textSecondary,
    fontWeight: '700',
    letterSpacing: 1,
    fontFamily: Font.mono,
  },
  card: {
    backgroundColor: Colors.surface,
    borderRadius: Radius.md,
    padding: Spacing.md,
    borderWidth: 1,
    borderColor: Colors.border,
    marginBottom: Spacing.md,
  },
  cardHeaderRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: Spacing.md,
  },
  cardHeader: {
    fontSize: 11,
    fontWeight: '800',
    color: Colors.textPrimary,
    letterSpacing: 1.5,
    fontFamily: Font.mono,
  },
  cardMeta: {
    fontSize: 9,
    fontWeight: '700',
    color: Colors.textMuted,
    letterSpacing: 1,
    fontFamily: Font.mono,
  },
  cardMetaSuccess: {
    fontSize: 9,
    fontWeight: '700',
    color: Colors.success,
    letterSpacing: 1,
    fontFamily: Font.mono,
  },
  cardSubtitle: {
    fontSize: 12,
    color: Colors.textSecondary,
    lineHeight: 18,
    marginBottom: Spacing.md,
  },
  formGroup: {
    marginBottom: Spacing.md,
  },
  label: {
    fontSize: 10,
    fontWeight: '800',
    color: Colors.textSecondary,
    letterSpacing: 1,
    marginBottom: 6,
    fontFamily: Font.mono,
  },
  input: {
    borderWidth: 1,
    borderColor: Colors.border,
    borderRadius: Radius.sm,
    paddingHorizontal: 12,
    paddingVertical: 10,
    fontSize: 13,
    color: Colors.textPrimary,
    backgroundColor: Colors.surfaceSecondary,
  },
  textArea: {
    height: 70,
    textAlignVertical: 'top',
  },
  photoPlaceholder: {
    height: 140,
    borderWidth: 1,
    borderColor: Colors.border,
    borderStyle: 'dashed',
    borderRadius: Radius.md,
    backgroundColor: Colors.surfaceSecondary,
    alignItems: 'center',
    justifyContent: 'center',
  },
  placeholderTitle: {
    fontSize: 12,
    fontWeight: '800',
    color: Colors.primary,
    marginTop: 6,
    letterSpacing: 1,
    fontFamily: Font.mono,
  },
  placeholderSub: {
    fontSize: 10,
    color: Colors.textMuted,
    marginTop: 2,
    fontFamily: Font.mono,
  },
  photoPreviewContainer: {
    position: 'relative',
    borderRadius: Radius.md,
    overflow: 'hidden',
  },
  photoPreview: {
    width: '100%',
    height: 200,
    borderRadius: Radius.md,
  },
  retakeBtn: {
    position: 'absolute',
    bottom: 10,
    right: 10,
    backgroundColor: Colors.chrome,
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
    paddingHorizontal: 10,
    paddingVertical: 6,
    borderRadius: Radius.sm,
  },
  retakeBtnText: {
    color: Colors.textOnChrome,
    fontSize: 10,
    fontWeight: '800',
    letterSpacing: 1,
    fontFamily: Font.mono,
  },
  submitBtn: {
    height: 48,
    backgroundColor: Colors.primary,
    borderRadius: Radius.md,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 8,
    marginTop: 8,
  },
  submitBtnDisabled: {
    opacity: 0.5,
    backgroundColor: Colors.border,
  },
  submitBtnText: {
    color: Colors.textOnPrimary,
    fontSize: 12,
    fontWeight: '800',
    letterSpacing: 2,
    fontFamily: Font.mono,
  },
  cameraContainer: {
    flex: 1,
    backgroundColor: Colors.chrome,
  },
  cameraView: {
    flex: 1,
  },
  cameraOverlay: {
    flex: 1,
    backgroundColor: 'rgba(0,0,0,0.5)',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingVertical: 40,
    paddingHorizontal: 20,
  },
  scannerFrame: {
    position: 'absolute',
    top: '30%',
    width: 200,
    height: 120,
    borderWidth: 2,
    borderColor: Colors.primary,
    backgroundColor: 'transparent',
  },
  cameraGuideText: {
    color: Colors.textOnChrome,
    fontSize: 11,
    fontWeight: '800',
    letterSpacing: 1.5,
    fontFamily: Font.mono,
    backgroundColor: Colors.chrome,
    paddingHorizontal: 12,
    paddingVertical: 6,
    borderRadius: Radius.sm,
    marginTop: 40,
  },
  captureBtn: {
    width: 60,
    height: 60,
    borderRadius: 30,
    borderWidth: 3,
    borderColor: Colors.textOnChrome,
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: 20,
  },
  captureInnerCircle: {
    width: 44,
    height: 44,
    borderRadius: 22,
    backgroundColor: Colors.textOnChrome,
  },
  closeCameraBtn: {
    position: 'absolute',
    top: 40,
    right: 20,
    padding: 10,
  },
  permissionBox: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: 30,
  },
  permissionText: {
    color: Colors.textOnChrome,
    fontSize: 13,
    textAlign: 'center',
    marginBottom: 20,
    lineHeight: 20,
  },
  permissionBtn: {
    backgroundColor: Colors.primary,
    paddingHorizontal: 20,
    paddingVertical: 12,
    borderRadius: Radius.md,
    marginBottom: 12,
  },
  permissionBtnText: {
    color: Colors.textOnPrimary,
    fontWeight: '800',
    fontSize: 11,
    letterSpacing: 1.5,
    fontFamily: Font.mono,
  },
  cancelCameraBtn: {
    padding: 10,
  },
  cancelCameraText: {
    color: Colors.textOnChromeMuted,
    fontSize: 11,
    letterSpacing: 1,
    fontFamily: Font.mono,
  },
});
