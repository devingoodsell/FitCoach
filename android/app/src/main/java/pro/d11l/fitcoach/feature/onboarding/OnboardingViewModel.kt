package pro.d11l.fitcoach.feature.onboarding

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import pro.d11l.fitcoach.core.network.DietPrefsDto
import pro.d11l.fitcoach.core.network.ExperienceDto
import pro.d11l.fitcoach.core.network.GoalWeightsDto
import pro.d11l.fitcoach.core.network.PreferencesDto
import pro.d11l.fitcoach.core.network.ProfileDto
import pro.d11l.fitcoach.core.network.ScheduleDto
import pro.d11l.fitcoach.data.OnboardingRepository
import pro.d11l.fitcoach.data.SaveResult

/** Wizard steps. Profile/Goals/Schedule are the required core path; Diet and
 *  Preferences are optional and skippable. */
enum class OnboardingStep { Intro, Profile, Goals, Schedule, Diet, Preferences, Done }

/** Estimated minutes for the core path, shown up front (E2-S1). */
const val ESTIMATED_MINUTES = 3

data class OnboardingUiState(
    val step: OnboardingStep = OnboardingStep.Intro,
    // profile
    val sex: String = "",
    val age: String = "",
    val heightCm: String = "",
    val weightKg: String = "",
    val level: String = "",
    // goals (relative weights; backend normalizes)
    val strength: Float = 0.5f,
    val healthspan: Float = 0.5f,
    val bodyComp: Float = 0.5f,
    val performance: Float = 0.5f,
    // schedule
    val daysPerWeek: String = "3",
    val sessionLengthMin: String = "60",
    // diet
    val dietPattern: String = "",
    val supplements: String = "",
    // preferences
    val likes: String = "",
    val dislikes: String = "",
    val hardAvoids: String = "",
    // conditional follow-up (routes to E7 later)
    val somethingBothering: Boolean = false,
    // ui
    val isSubmitting: Boolean = false,
    val fieldErrors: Map<String, String> = emptyMap(),
    val error: String? = null,
    val completed: Boolean = false,
)

class OnboardingViewModel(private val repo: OnboardingRepository) : ViewModel() {

    private val _state = MutableStateFlow(OnboardingUiState())
    val state: StateFlow<OnboardingUiState> = _state.asStateFlow()

    // --- field events ---
    fun onSex(v: String) = _state.update { it.copy(sex = v, fieldErrors = it.fieldErrors - "sex") }
    fun onAge(v: String) = _state.update { it.copy(age = v.filter(Char::isDigit), fieldErrors = it.fieldErrors - "age") }
    fun onHeight(v: String) = _state.update { it.copy(heightCm = v) }
    fun onWeight(v: String) = _state.update { it.copy(weightKg = v) }
    fun onLevel(v: String) = _state.update { it.copy(level = v, fieldErrors = it.fieldErrors - "experience.level") }
    fun onStrength(v: Float) = _state.update { it.copy(strength = v) }
    fun onHealthspan(v: Float) = _state.update { it.copy(healthspan = v) }
    fun onBodyComp(v: Float) = _state.update { it.copy(bodyComp = v) }
    fun onPerformance(v: Float) = _state.update { it.copy(performance = v) }
    fun onDaysPerWeek(v: String) = _state.update { it.copy(daysPerWeek = v.filter(Char::isDigit)) }
    fun onSessionLength(v: String) = _state.update { it.copy(sessionLengthMin = v.filter(Char::isDigit)) }
    fun onDietPattern(v: String) = _state.update { it.copy(dietPattern = v, fieldErrors = it.fieldErrors - "pattern") }
    fun onSupplements(v: String) = _state.update { it.copy(supplements = v) }
    fun onLikes(v: String) = _state.update { it.copy(likes = v) }
    fun onDislikes(v: String) = _state.update { it.copy(dislikes = v) }
    fun onHardAvoids(v: String) = _state.update { it.copy(hardAvoids = v) }
    fun onSomethingBothering(v: Boolean) = _state.update { it.copy(somethingBothering = v) }

    fun start() = goTo(OnboardingStep.Profile)
    fun finish() = _state.update { it.copy(completed = true) }

    fun back() {
        val prev = when (_state.value.step) {
            OnboardingStep.Profile -> OnboardingStep.Intro
            OnboardingStep.Goals -> OnboardingStep.Profile
            OnboardingStep.Schedule -> OnboardingStep.Goals
            OnboardingStep.Diet -> OnboardingStep.Schedule
            OnboardingStep.Preferences -> OnboardingStep.Diet
            else -> _state.value.step
        }
        goTo(prev)
    }

    fun skipDiet() = goTo(OnboardingStep.Preferences)
    fun skipPreferences() = goTo(OnboardingStep.Done)

    fun submitProfile() {
        val s = _state.value
        val errors = buildMap {
            if (s.sex.isBlank()) put("sex", "Select your sex")
            val age = s.age.toIntOrNull()
            if (age == null || age < 13 || age > 120) put("age", "Enter an age between 13 and 120")
            if (s.level.isBlank()) put("experience.level", "Select your experience level")
        }
        if (errors.isNotEmpty()) {
            _state.update { it.copy(fieldErrors = errors) }
            return
        }
        val dto = ProfileDto(
            age = s.age.toIntOrNull(),
            sex = s.sex,
            heightCm = s.heightCm.toDoubleOrNull(),
            weightKg = s.weightKg.toDoubleOrNull(),
            experience = ExperienceDto(level = s.level),
        )
        saveAndAdvance(OnboardingStep.Goals) { repo.saveProfile(dto) }
    }

    fun submitGoals() {
        val s = _state.value
        if (s.strength + s.healthspan + s.bodyComp + s.performance <= 0f) {
            _state.update { it.copy(error = "Set at least one goal above zero.") }
            return
        }
        val dto = GoalWeightsDto(
            strength = s.strength.toDouble(),
            healthspan = s.healthspan.toDouble(),
            bodyComposition = s.bodyComp.toDouble(),
            performance = s.performance.toDouble(),
        )
        saveAndAdvance(OnboardingStep.Schedule) { repo.saveGoals(dto) }
    }

    fun submitSchedule() {
        val s = _state.value
        val days = s.daysPerWeek.toIntOrNull()
        val len = s.sessionLengthMin.toIntOrNull()
        val errors = buildMap {
            if (days == null || days < 1 || days > 7) put("days_per_week", "Choose 1 to 7 days")
            if (len == null || len < 10 || len > 240) put("session_length_min", "Choose 10 to 240 minutes")
        }
        if (errors.isNotEmpty()) {
            _state.update { it.copy(fieldErrors = errors) }
            return
        }
        saveAndAdvance(OnboardingStep.Diet) { repo.saveSchedule(ScheduleDto(days!!, len!!)) }
    }

    fun submitDiet() {
        val s = _state.value
        if (s.dietPattern.isBlank()) {
            _state.update { it.copy(fieldErrors = mapOf("pattern" to "Choose a pattern or skip this step")) }
            return
        }
        saveAndAdvance(OnboardingStep.Preferences) { repo.saveDiet(DietPrefsDto(s.dietPattern, s.supplements)) }
    }

    fun submitPreferences() {
        val s = _state.value
        val dto = PreferencesDto(parseList(s.likes), parseList(s.dislikes), parseList(s.hardAvoids))
        saveAndAdvance(OnboardingStep.Done) { repo.savePreferences(dto) }
    }

    private fun saveAndAdvance(to: OnboardingStep, op: suspend () -> SaveResult) {
        if (_state.value.isSubmitting) return
        _state.update { it.copy(isSubmitting = true, error = null, fieldErrors = emptyMap()) }
        viewModelScope.launch {
            when (val result = op()) {
                is SaveResult.Ok -> _state.update { it.copy(isSubmitting = false, step = to) }
                is SaveResult.Invalid -> _state.update { it.copy(isSubmitting = false, fieldErrors = result.fields) }
                is SaveResult.Error -> _state.update { it.copy(isSubmitting = false, error = result.message) }
            }
        }
    }

    private fun goTo(step: OnboardingStep) =
        _state.update { it.copy(step = step, fieldErrors = emptyMap(), error = null) }

    private fun parseList(raw: String): List<String> =
        raw.split(",").map(String::trim).filter(String::isNotEmpty)
}
