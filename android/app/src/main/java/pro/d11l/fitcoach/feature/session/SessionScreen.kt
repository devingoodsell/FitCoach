package pro.d11l.fitcoach.feature.session

import android.content.Context
import android.media.AudioManager
import android.media.ToneGenerator
import android.os.Build
import android.os.VibrationEffect
import android.os.Vibrator
import android.os.VibratorManager
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import pro.d11l.fitcoach.core.designsystem.MedicalDisclaimer
import pro.d11l.fitcoach.data.PlanSet

@Composable
fun SessionScreen(viewModel: SessionViewModel, onBack: () -> Unit) {
    val state by viewModel.state.collectAsStateWithLifecycle()
    val context = LocalContext.current

    // Audible + haptic cue when a rest hits zero (E6-PR3).
    LaunchedEffect(state.restCueId) {
        if (state.restCueId > 0) playRestCue(context)
    }

    Column(
        modifier = Modifier.fillMaxSize().padding(24.dp).verticalScroll(rememberScrollState()),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        Text("Today's workout", style = MaterialTheme.typography.headlineSmall)
        state.error?.let { Text(it, color = MaterialTheme.colorScheme.error) }

        when {
            state.loading -> Text("Building your session…", style = MaterialTheme.typography.bodyMedium)
            state.plan == null ->
                Button(onClick = viewModel::start, modifier = Modifier.fillMaxWidth()) {
                    Text("Start workout")
                }
            state.finished -> SessionCompleteCard(state.loggedCount, state.totalSteps)
            else -> PlayerContent(state, viewModel)
        }

        MedicalDisclaimer(modifier = Modifier.fillMaxWidth())
        OutlinedButton(onClick = onBack, modifier = Modifier.fillMaxWidth()) { Text("Back") }
    }
}

@Composable
private fun PlayerContent(state: SessionUiState, vm: SessionViewModel) {
    val step = state.current ?: return

    Text(
        "Set ${state.loggedCount + 1} of ${state.totalSteps}",
        style = MaterialTheme.typography.labelLarge,
    )
    CurrentSetCard(step)
    state.rest?.let { RestPanel(it, vm) }
    SetInputs(state, step, vm)
}

@Composable
private fun CurrentSetCard(step: PlanSet) {
    Card(modifier = Modifier.fillMaxWidth()) {
        Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
            Text(step.blockTitle, style = MaterialTheme.typography.labelMedium)
            Text(step.exerciseName, style = MaterialTheme.typography.titleLarge)
            Text(
                "Set ${step.setIndexInExercise + 1} of ${step.setCountInExercise}",
                style = MaterialTheme.typography.bodySmall,
            )
            Text("Target: ${formatPrescription(step.prescription)}", style = MaterialTheme.typography.bodyMedium)
            step.notes?.takeIf { it.isNotBlank() }?.let {
                Text(it, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
        }
    }
}

@Composable
private fun SetInputs(state: SessionUiState, step: PlanSet, vm: SessionViewModel) {
    val isTimed = step.prescription.type == "time"
    Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
        if (isTimed) {
            OutlinedTextField(
                value = state.draft.durationSec,
                onValueChange = vm::updateDuration,
                label = { Text("Seconds") },
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                modifier = Modifier.fillMaxWidth(),
            )
        } else {
            Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                OutlinedTextField(
                    value = state.draft.reps,
                    onValueChange = vm::updateReps,
                    label = { Text("Reps") },
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                    modifier = Modifier.weight(1f),
                )
                OutlinedTextField(
                    value = state.draft.loadKg,
                    onValueChange = vm::updateLoad,
                    label = { Text("Weight (kg)") },
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                    modifier = Modifier.weight(1f),
                )
            }
        }
        Button(onClick = vm::logCurrentSet, modifier = Modifier.fillMaxWidth()) { Text("Log set") }
        OutlinedButton(onClick = vm::skipCurrentSet, modifier = Modifier.fillMaxWidth()) { Text("Skip set") }
    }
}

@Composable
private fun RestPanel(rest: RestState, vm: SessionViewModel) {
    Card(modifier = Modifier.fillMaxWidth()) {
        Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            if (rest.finished) {
                Text("Rest done", style = MaterialTheme.typography.titleMedium)
                TextButton(onClick = vm::dismissRest) { Text("Dismiss") }
            } else {
                Text("Rest ${formatClock(rest.remainingSec)}", style = MaterialTheme.typography.titleMedium)
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    if (rest.running) {
                        TextButton(onClick = vm::pauseRest) { Text("Pause") }
                    } else {
                        TextButton(onClick = vm::resumeRest) { Text("Resume") }
                    }
                    TextButton(onClick = vm::extendRest) { Text("+15s") }
                    TextButton(onClick = vm::skipRest) { Text("Skip rest") }
                }
            }
        }
    }
}

@Composable
private fun SessionCompleteCard(logged: Int, total: Int) {
    Card(modifier = Modifier.fillMaxWidth()) {
        Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text("Session complete", style = MaterialTheme.typography.titleLarge)
            Text("Logged $logged of $total sets.", style = MaterialTheme.typography.bodyMedium)
        }
    }
}

private fun formatClock(totalSec: Int): String = "%d:%02d".format(totalSec / 60, totalSec % 60)

/** Plays a short beep and a vibration when a rest ends (E6-PR3). Offline. */
private fun playRestCue(context: Context) {
    runCatching {
        ToneGenerator(AudioManager.STREAM_ALARM, 80).apply {
            startTone(ToneGenerator.TONE_PROP_BEEP, 250)
        }
    }
    runCatching {
        val vibrator = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
            val manager = context.getSystemService(Context.VIBRATOR_MANAGER_SERVICE) as VibratorManager
            manager.defaultVibrator
        } else {
            @Suppress("DEPRECATION")
            context.getSystemService(Context.VIBRATOR_SERVICE) as Vibrator
        }
        vibrator.vibrate(VibrationEffect.createOneShot(400, VibrationEffect.DEFAULT_AMPLITUDE))
    }
}
