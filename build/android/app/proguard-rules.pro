# Add project specific ProGuard rules here.
# You can control the set of applied configuration files using the
# proguardFiles setting in build.gradle.

# Keep native methods
-keepclasseswithmembernames class * {
    native <methods>;
}

# Keep Wails bridge classes
-keep class com.wails.app.WailsBridge { *; }
-keep class com.wails.app.WailsJSBridge { *; }

# Optional annotations referenced by Tink (via androidx.security.crypto); absent from classpath
-dontwarn javax.annotation.Nullable
-dontwarn javax.annotation.concurrent.GuardedBy
