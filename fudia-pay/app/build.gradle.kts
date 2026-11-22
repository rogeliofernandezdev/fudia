plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.android")
    id("org.jetbrains.kotlin.plugin.compose")
    id("org.jetbrains.kotlin.plugin.serialization")
    id("com.google.devtools.ksp")
    id("com.google.dagger.hilt.android")
}

android {
    namespace = "com.fudia.pay"
    compileSdk = 35

    defaultConfig {
        applicationId = "com.fudia.pay"
        minSdk = 24
        targetSdk = 35
        versionCode = 1
        versionName = "0.1.0"
        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"

        // Push de Firebase Cloud Messaging que despierta la app cuando el
        // fabricante la congela (HONOR/MagicOS). Se configura al compilar con
        // -PfcmProjectId, -PfcmAppId, -PfcmApiKey y -PfcmSenderId (consola de
        // Firebase › app Android com.fudia.pay); sin ellos la app funciona solo
        // con su alarma de respaldo. No se versiona google-services.json.
        fun fcm(name: String) = "\"${project.findProperty(name) ?: ""}\""
        buildConfigField("String", "FCM_PROJECT_ID", fcm("fcmProjectId"))
        buildConfigField("String", "FCM_APP_ID", fcm("fcmAppId"))
        buildConfigField("String", "FCM_API_KEY", fcm("fcmApiKey"))
        buildConfigField("String", "FCM_SENDER_ID", fcm("fcmSenderId"))
    }

    // El backend por variante: la URL se inyecta al compilar con
    // -PfoodsApiUrl=https://… ; sin ella, debug usa el entorno de desarrollo y
    // release el de producción. Nunca se configura en la pantalla del
    // restaurante ni se fija fuera de aquí.
    buildTypes {
        debug {
            buildConfigField("String", "FOODS_API_URL",
                "\"${project.findProperty("foodsApiUrl") ?: "https://foods-backend-dev.a.run.app/"}\"")
        }
        release {
            isMinifyEnabled = true
            proguardFiles(getDefaultProguardFile("proguard-android-optimize.txt"), "proguard-rules.pro")
            buildConfigField("String", "FOODS_API_URL",
                "\"${project.findProperty("foodsApiUrl") ?: "https://foods-backend.a.run.app/"}\"")
        }
    }
    buildFeatures {
        compose = true
        buildConfig = true
    }
    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
}

kotlin {
    jvmToolchain(17)
}

dependencies {
    val composeBom = platform("androidx.compose:compose-bom:2024.12.01")
    implementation(composeBom)

    implementation("androidx.core:core-ktx:1.15.0")
    implementation("androidx.activity:activity-compose:1.9.3")
    implementation("androidx.lifecycle:lifecycle-runtime-compose:2.8.7")
    implementation("androidx.lifecycle:lifecycle-viewmodel-compose:2.8.7")
    implementation("androidx.compose.ui:ui")
    implementation("androidx.compose.ui:ui-tooling-preview")
    implementation("androidx.compose.material3:material3")
    implementation("androidx.navigation:navigation-compose:2.8.5")
    implementation("androidx.hilt:hilt-navigation-compose:1.2.0")

    implementation("com.google.dagger:hilt-android:2.52")
    ksp("com.google.dagger:hilt-compiler:2.52")
    implementation("androidx.hilt:hilt-work:1.2.0")
    ksp("androidx.hilt:hilt-compiler:1.2.0")

    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-android:1.9.0")
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-json:1.7.3")

    implementation("com.squareup.retrofit2:retrofit:2.11.0")
    implementation("com.squareup.retrofit2:converter-kotlinx-serialization:2.11.0")
    implementation("com.squareup.okhttp3:okhttp:4.12.0")

    implementation("androidx.datastore:datastore-preferences:1.1.1")
    implementation("androidx.security:security-crypto:1.1.0-alpha06")

    implementation("androidx.room:room-runtime:2.6.1")
    implementation("androidx.room:room-ktx:2.6.1")
    ksp("androidx.room:room-compiler:2.6.1")

    implementation("androidx.work:work-runtime-ktx:2.10.0")

    // Solo mensajería (push de datos); sin Analytics ni google-services.
    implementation(platform("com.google.firebase:firebase-bom:33.7.0"))
    implementation("com.google.firebase:firebase-messaging")

    implementation("com.journeyapps:zxing-android-embedded:4.3.0")
    implementation("com.google.zxing:core:3.5.3")

    testImplementation("junit:junit:4.13.2")
    testImplementation("org.jetbrains.kotlinx:kotlinx-coroutines-test:1.9.0")
    testImplementation("com.squareup.okhttp3:mockwebserver:4.12.0")

    androidTestImplementation("androidx.test.ext:junit:1.2.1")
    androidTestImplementation(composeBom)
    androidTestImplementation("androidx.compose.ui:ui-test-junit4")
}
