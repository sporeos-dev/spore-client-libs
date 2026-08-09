#include <gtest/gtest.h>

// Change to run a subset, e.g. "Fixture.*" or "Fixture.CreateDestroy"
// Overridden by --gtest_filter=... on the command line, or make test FILTER=...
static const char* kFilter = "*";

int main(int argc, char** argv)
{
    testing::InitGoogleTest(&argc, argv);
    if (testing::GTEST_FLAG(filter) == "*")
        testing::GTEST_FLAG(filter) = kFilter;
    return RUN_ALL_TESTS();
}
