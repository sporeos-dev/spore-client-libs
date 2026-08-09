#pragma once

#include <gtest/gtest.h>
#include "spore_c.h"

class Fixture : public ::testing::Test
{
protected:
    virtual ~Fixture() = default;

    spore_client_t* client = nullptr;

    void SetUp() override
    {
        client = spore_client_create("test.node");
    }

    void TearDown() override
    {
        spore_client_destroy(client);
        client = nullptr;
    }
};
