package com.fudia.pay.core.design

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color

/** Paleta de Fudia, tomada del sistema de diseño de foods-admin-web:
 * azul operativo para acción, marino para identidad, verde restaurante para
 * lo confirmado, superficies claras. Nada de tema por defecto. */
val FudiaPrimary = Color(0xFF0052FC) // primary-600: acción
val FudiaPrimaryDark = Color(0xFF0043CF) // primary-700: hover y texto de acción
val FudiaNavy = Color(0xFF052352) // navigation-900: identidad
val FudiaNavyDeep = Color(0xFF041A40) // navigation-950
val FudiaNavigationText = Color(0xFFC4D3EE) // navigation-text: descripción sobre marino
val FudiaCloud = Color(0xFFF4F7FD) // cloud-50: fondo
val FudiaCloudLine = Color(0xFFE1E7F2) // line
val FudiaGreen = Color(0xFF03C056) // brand-600: confirmado
val FudiaGreenDark = Color(0xFF007B35) // brand-700: legible
val FudiaError = Color(0xFFBF2817) // danger-600
val FudiaInk = Color(0xFF052352) // la tinta comparte la familia del marino

private val FudiaColorScheme = lightColorScheme(
    primary = FudiaPrimary,
    onPrimary = Color.White,
    primaryContainer = Color(0xFFEAF0FF),
    onPrimaryContainer = FudiaPrimaryDark,
    secondary = FudiaGreen,
    onSecondary = Color.White,
    background = FudiaCloud,
    onBackground = FudiaInk,
    surface = Color.White,
    onSurface = FudiaInk,
    surfaceVariant = Color(0xFFF0F2F8),
    onSurfaceVariant = Color(0xFF39507A),
    error = FudiaError,
    onError = Color.White,
    outline = FudiaCloudLine,
)

@Composable
fun FudiaTheme(content: @Composable () -> Unit) {
    MaterialTheme(colorScheme = FudiaColorScheme, content = content)
}
