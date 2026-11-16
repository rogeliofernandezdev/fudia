# Reglas de ofuscación: se conservan los genéricos y serializers que Retrofit y
# kotlinx-serialization leen por reflexión; el resto se reduce con normalidad.

-keepattributes Signature, InnerClasses, EnclosingMethod, *Annotation*

-keep,includedescriptorclasses class com.fudia.pay.**$$serializer { *; }
-keep,includedescriptorclasses interface com.fudia.pay.**$$serializer
-keepclassmembers class com.fudia.pay.** {
    *** Companion;
}
-keepclasseswithmembers class com.fudia.pay.** {
    kotlinx.serialization.KSerializer serializer(...);
}

-dontwarn okhttp3.internal.platform.**
-dontwarn org.conscrypt.**
-dontwarn org.bouncycastle.**
-dontwarn org.openjsse.**
