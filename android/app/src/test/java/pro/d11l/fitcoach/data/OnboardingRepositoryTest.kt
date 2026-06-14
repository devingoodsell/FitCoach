package pro.d11l.fitcoach.data

import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import pro.d11l.fitcoach.core.network.ExperienceDto
import pro.d11l.fitcoach.core.network.ProfileDto
import pro.d11l.fitcoach.testing.FakeApi
import pro.d11l.fitcoach.testing.validationErrorResponse

class OnboardingRepositoryTest {

    private val validProfile = ProfileDto(age = 30, sex = "male", experience = ExperienceDto(level = "novice"))

    @Test
    fun `successful save returns Ok`() = runTest {
        val repo = OnboardingRepository(FakeApi())
        assertTrue(repo.saveProfile(validProfile) is SaveResult.Ok)
    }

    @Test
    fun `400 response parses field errors`() = runTest {
        val api = FakeApi().apply {
            profileResponse = validationErrorResponse<ProfileDto>(mapOf("sex" to "required"))
        }
        val result = repo(api).saveProfile(validProfile)
        assertTrue(result is SaveResult.Invalid)
        assertEquals("required", (result as SaveResult.Invalid).fields["sex"])
    }

    private fun repo(api: FakeApi) = OnboardingRepository(api)
}
